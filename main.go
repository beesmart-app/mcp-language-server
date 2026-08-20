package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/beesmart-app/mcp-language-server/internal/logging"
	"github.com/beesmart-app/mcp-language-server/internal/lsp"
	"github.com/beesmart-app/mcp-language-server/internal/watcher"
	"github.com/mark3labs/mcp-go/server"
)

// Create a logger for the core component
var coreLogger = logging.NewLogger(logging.Core)

type config struct {
	workspaceDir string
	lspCommand   string
	lspArgs      []string
	listenSocket string
	idleTimeout  time.Duration
}

type mcpServer struct {
	config           config
	lspClient        *lsp.Client
	mcpServer        *server.MCPServer
	ctx              context.Context
	cancelFunc       context.CancelFunc
	workspaceWatcher *watcher.WorkspaceWatcher

	// Estado do modo daemon (config.listenSocket != ""): contador de sessoes
	// SSE ativas e timer de encerramento por ociosidade. Nao usado no modo
	// stdio de sessao unica (1 processo por sessao, sem conceito de "sessao
	// ativa" separado do proprio processo).
	idleShutdown chan struct{}
	sessionMu    sync.Mutex
	activeCount  int
	idleTimer    *time.Timer
}

func parseConfig() (*config, error) {
	cfg := &config{}
	flag.StringVar(&cfg.workspaceDir, "workspace", "", "Path to workspace directory")
	flag.StringVar(&cfg.lspCommand, "lsp", "", "LSP command to run (args should be passed after --)")
	flag.StringVar(&cfg.listenSocket, "listen", "", "Unix socket path to serve MCP over SSE for multiple concurrent clients (daemon mode). Empty (default): serve a single client over stdio, as before.")
	flag.DurationVar(&cfg.idleTimeout, "idle-timeout", 20*time.Minute, "Daemon mode only (-listen): shut down after this long with zero connected sessions.")
	flag.Parse()

	// Get remaining args after -- as LSP arguments
	cfg.lspArgs = flag.Args()

	// Validate workspace directory
	if cfg.workspaceDir == "" {
		return nil, fmt.Errorf("workspace directory is required")
	}

	workspaceDir, err := filepath.Abs(cfg.workspaceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for workspace: %v", err)
	}
	cfg.workspaceDir = workspaceDir

	if _, err := os.Stat(cfg.workspaceDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("workspace directory does not exist: %s", cfg.workspaceDir)
	}

	// Validate LSP command
	if cfg.lspCommand == "" {
		return nil, fmt.Errorf("LSP command is required")
	}

	if _, err := exec.LookPath(cfg.lspCommand); err != nil {
		return nil, fmt.Errorf("LSP command not found: %s", cfg.lspCommand)
	}

	return cfg, nil
}

func newServer(config *config) (*mcpServer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &mcpServer{
		config:       *config,
		ctx:          ctx,
		cancelFunc:   cancel,
		idleShutdown: make(chan struct{}),
	}, nil
}

func (s *mcpServer) initializeLSP() error {
	if err := os.Chdir(s.config.workspaceDir); err != nil {
		return fmt.Errorf("failed to change to workspace directory: %v", err)
	}

	client, err := lsp.NewClient(s.config.lspCommand, s.config.lspArgs...)
	if err != nil {
		return fmt.Errorf("failed to create LSP client: %v", err)
	}
	s.lspClient = client
	s.workspaceWatcher = watcher.NewWorkspaceWatcher(client)

	initResult, err := client.InitializeLSPClient(s.ctx, s.config.workspaceDir)
	if err != nil {
		return fmt.Errorf("initialize failed: %v", err)
	}

	coreLogger.Debug("Server capabilities: %+v", initResult.Capabilities)

	go s.workspaceWatcher.WatchWorkspace(s.ctx, s.config.workspaceDir)
	return client.WaitForServerReady(s.ctx)
}

func (s *mcpServer) start() error {
	if err := s.initializeLSP(); err != nil {
		return err
	}

	opts := []server.ServerOption{
		server.WithLogging(),
		server.WithRecovery(),
	}
	if s.config.listenSocket != "" {
		opts = append(opts, server.WithHooks(s.idleTimeoutHooks()))
	}
	s.mcpServer = server.NewMCPServer("MCP Language Server", "v0.0.2", opts...)

	err := s.registerTools()
	if err != nil {
		return fmt.Errorf("tool registration failed: %v", err)
	}

	if s.config.listenSocket != "" {
		return s.serveDaemon()
	}
	return server.ServeStdio(s.mcpServer)
}

// serveDaemon serve MCP sobre SSE num socket Unix, permitindo multiplas
// sessoes (bridges stdio de varias sessoes Claude Code) compartilharem o
// mesmo lsp.Client/jdtls. baseURL "http://unix" e so um placeholder valido
// pra montar o endpoint de mensagem absoluto que o SSEServer manda pro
// cliente - quem consome esse endpoint (cmd/bridge) ignora o host real e
// disca sempre no socket via DialContext customizado.
func (s *mcpServer) serveDaemon() error {
	_ = os.Remove(s.config.listenSocket)
	listener, err := net.Listen("unix", s.config.listenSocket)
	if err != nil {
		return fmt.Errorf("failed to listen on unix socket %s: %v", s.config.listenSocket, err)
	}
	defer listener.Close()
	defer os.Remove(s.config.listenSocket)

	sseServer := server.NewSSEServer(s.mcpServer, server.WithBaseURL("http://unix"))
	coreLogger.Info("Serving MCP over SSE on unix socket: %s (idle timeout: %s)", s.config.listenSocket, s.config.idleTimeout)
	return http.Serve(listener, sseServer)
}

// idleTimeoutHooks arma/desarma um timer de encerramento por ociosidade:
// zera o timer quando uma sessao entra, arma quando a ultima sessao sai.
// So faz sentido no modo daemon (sessao == conexao SSE de um bridge) - no
// modo stdio de sessao unica o proprio processo morre com a sessao, sem
// precisar de timer.
func (s *mcpServer) idleTimeoutHooks() *server.Hooks {
	hooks := &server.Hooks{}
	hooks.AddOnRegisterSession(func(ctx context.Context, session server.ClientSession) {
		s.sessionMu.Lock()
		defer s.sessionMu.Unlock()
		s.activeCount++
		if s.idleTimer != nil {
			s.idleTimer.Stop()
			s.idleTimer = nil
		}
		coreLogger.Info("Session registered (%s), active=%d", session.SessionID(), s.activeCount)
	})
	hooks.AddOnUnregisterSession(func(ctx context.Context, session server.ClientSession) {
		s.sessionMu.Lock()
		defer s.sessionMu.Unlock()
		if s.activeCount > 0 {
			s.activeCount--
		}
		coreLogger.Info("Session unregistered (%s), active=%d", session.SessionID(), s.activeCount)
		if s.activeCount == 0 {
			if s.idleTimer != nil {
				s.idleTimer.Stop()
			}
			idleTimeout := s.config.idleTimeout
			s.idleTimer = time.AfterFunc(idleTimeout, func() {
				coreLogger.Info("Idle timeout (%s) reached with no active sessions, shutting down", idleTimeout)
				select {
				case <-s.idleShutdown: // ja fechado
				default:
					close(s.idleShutdown)
				}
			})
		}
	})
	return hooks
}

func main() {
	coreLogger.Info("MCP Language Server starting")

	done := make(chan struct{})
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	config, err := parseConfig()
	if err != nil {
		coreLogger.Fatal("%v", err)
	}

	server, err := newServer(config)
	if err != nil {
		coreLogger.Fatal("%v", err)
	}

	// Parent process monitoring channel
	parentDeath := make(chan struct{})

	// Monitor parent process termination
	// Claude desktop does not properly kill child processes for MCP servers.
	// So faz sentido pra sessao stdio unica (1 processo por sessao) - um
	// daemon (-listen) deve sobreviver a qualquer sessao individual que o
	// tenha iniciado, entao esse monitor fica desligado nesse modo.
	if config.listenSocket == "" {
		go func() {
			ppid := os.Getppid()
			coreLogger.Debug("Monitoring parent process: %d", ppid)

			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					currentPpid := os.Getppid()
					if currentPpid != ppid && (currentPpid == 1 || ppid == 1) {
						coreLogger.Info("Parent process %d terminated (current ppid: %d), initiating shutdown", ppid, currentPpid)
						close(parentDeath)
						return
					}
				case <-done:
					return
				}
			}
		}()
	}

	// Handle shutdown triggers. cleanup() so encerra o lado LSP (fecha
	// arquivos, manda shutdown/exit pro jdtls) - nao faz o processo em si
	// terminar. No modo stdio original isso "funcionava" so porque
	// ServeStdio costuma desbloquear/o processo eventualmente leva SIGKILL
	// de quem gerencia (comentario acima ja apontava que o Claude
	// desktop/Code nao mata direito processos filhos de MCP server). No
	// modo daemon isso e um bug real: http.Serve nunca desbloqueia sozinho,
	// entao sem os.Exit explicito aqui o processo fica vivo pra sempre
	// segurando o lock/socket mesmo depois do idle-timeout "completar".
	// Chamar os.Exit aqui (em vez de confiar no fluxo sequencial abaixo
	// alcancar <-done) garante saida em qualquer um dos tres modos.
	go func() {
		select {
		case sig := <-sigChan:
			coreLogger.Info("Received signal %v in PID: %d", sig, os.Getpid())
			cleanup(server, done)
		case <-server.idleShutdown:
			coreLogger.Info("Idle shutdown triggered")
			cleanup(server, done)
		case <-parentDeath:
			coreLogger.Info("Parent death detected, initiating shutdown")
			cleanup(server, done)
		}
		coreLogger.Info("Server shutdown complete for PID: %d", os.Getpid())
		os.Exit(0)
	}()

	if err := server.start(); err != nil {
		coreLogger.Error("Server error: %v", err)
		cleanup(server, done)
		os.Exit(1)
	}

	<-done
	coreLogger.Info("Server shutdown complete for PID: %d", os.Getpid())
	os.Exit(0)
}

func cleanup(s *mcpServer, done chan struct{}) {
	coreLogger.Info("Cleanup initiated for PID: %d", os.Getpid())

	// Create a context with timeout for shutdown operations
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if s.lspClient != nil {
		coreLogger.Info("Closing open files")
		s.lspClient.CloseAllFiles(ctx)

		// Create a shorter timeout context for the shutdown request
		shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 500*time.Millisecond)
		defer shutdownCancel()

		// Run shutdown in a goroutine with timeout to avoid blocking if LSP doesn't respond
		shutdownDone := make(chan struct{})
		go func() {
			coreLogger.Info("Sending shutdown request")
			if err := s.lspClient.Shutdown(shutdownCtx); err != nil {
				coreLogger.Error("Shutdown request failed: %v", err)
			}
			close(shutdownDone)
		}()

		// Wait for shutdown with timeout
		select {
		case <-shutdownDone:
			coreLogger.Info("Shutdown request completed")
		case <-time.After(1 * time.Second):
			coreLogger.Warn("Shutdown request timed out, proceeding with exit")
		}

		coreLogger.Info("Sending exit notification")
		if err := s.lspClient.Exit(ctx); err != nil {
			coreLogger.Error("Exit notification failed: %v", err)
		}

		coreLogger.Info("Closing LSP client")
		if err := s.lspClient.Close(); err != nil {
			coreLogger.Error("Failed to close LSP client: %v", err)
		}
	}

	// Send signal to the done channel
	select {
	case <-done: // Channel already closed
	default:
		close(done)
	}

	coreLogger.Info("Cleanup completed for PID: %d", os.Getpid())
}
