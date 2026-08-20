// Command bridge traduz MCP-sobre-stdio (o que o Claude Code sabe falar com
// um "command" no .mcp.json) para MCP-sobre-SSE contra um socket Unix (o
// modo daemon do mcp-language-server, ver -listen em main.go). Generico: nao
// sabe nada de jdtls/Java, so espelha as tools do servidor remoto.
//
// Uso: bridge -socket /caminho/para/daemon.sock
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	socketPath := flag.String("socket", "", "Caminho do socket Unix do daemon (mcp-language-server -listen)")
	connectTimeout := flag.Duration("connect-timeout", 30*time.Second, "Tempo maximo esperando o socket do daemon aparecer/responder")
	flag.Parse()

	if *socketPath == "" {
		fmt.Fprintln(os.Stderr, "bridge: -socket e obrigatorio")
		os.Exit(1)
	}

	ctx := context.Background()

	daemonClient, err := dialDaemon(ctx, *socketPath, *connectTimeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bridge: falha conectando no daemon (%s): %v\n", *socketPath, err)
		os.Exit(1)
	}
	defer daemonClient.Close()

	toolsResult, err := daemonClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "bridge: falha listando tools do daemon: %v\n", err)
		os.Exit(1)
	}

	localServer := server.NewMCPServer("mcp-language-server-bridge", "v1.0.0")
	for _, tool := range toolsResult.Tools {
		tool := tool
		localServer.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return daemonClient.CallTool(ctx, request)
		})
	}

	if err := server.ServeStdio(localServer); err != nil {
		fmt.Fprintf(os.Stderr, "bridge: erro servindo stdio: %v\n", err)
		os.Exit(1)
	}
}

// dialDaemon conecta no daemon via SSE sobre um socket Unix. O baseURL e um
// placeholder valido (precisa de scheme http(s) e host nao-vazio pra passar
// na validacao da lib) - o host real e ignorado pelo DialContext customizado,
// que sempre disca o socket Unix independente do que a URL diz.
func dialDaemon(ctx context.Context, socketPath string, timeout time.Duration) (*client.Client, error) {
	dialer := &net.Dialer{}
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return dialer.DialContext(ctx, "unix", socketPath)
			},
		},
	}

	c, err := client.NewSSEMCPClient("http://unix/sse", client.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("criando client SSE: %w", err)
	}

	connectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// O socket pode ainda nao existir se o daemon estiver subindo agora
	// mesmo (ensure-script acabou de spawnar) - tenta com backoff curto em
	// vez de falhar na primeira tentativa. IMPORTANTE: c.Start precisa do
	// ctx de vida longa (nao o connectCtx, que e cancelado no fim desta
	// funcao) - o transporte SSE deriva DESSE ctx o context.WithCancel que
	// governa a goroutine de leitura do stream inteiro (client/transport/
	// sse.go: Start faz "ctx, cancel := context.WithCancel(ctx)" e usa esse
	// ctx no request HTTP de longa duracao). Passar connectCtx aqui derruba
	// o stream assim que dialDaemon retorna (defer cancel() acima).
	var startErr error
	for {
		startErr = c.Start(ctx)
		if startErr == nil {
			break
		}
		select {
		case <-connectCtx.Done():
			return nil, fmt.Errorf("timeout conectando: ultimo erro: %w", startErr)
		case <-time.After(300 * time.Millisecond):
		}
	}

	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "mcp-language-server-bridge",
		Version: "v1.0.0",
	}
	_, err = c.Initialize(connectCtx, initReq)
	if err != nil {
		return nil, fmt.Errorf("initialize: %w", err)
	}

	return c, nil
}
