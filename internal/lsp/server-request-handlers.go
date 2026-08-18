package lsp

import (
	"encoding/json"

	"github.com/beesmart-app/mcp-language-server/internal/protocol"
	"github.com/beesmart-app/mcp-language-server/internal/utilities"
)

// FileWatchHandler is called when file watchers are registered by the server
type FileWatchHandler func(id string, watchers []protocol.FileSystemWatcher)

// fileWatchHandler holds the current file watch handler
var fileWatchHandler FileWatchHandler

// RegisterFileWatchHandler registers a handler for file watcher registrations
func RegisterFileWatchHandler(handler FileWatchHandler) {
	fileWatchHandler = handler
}

// Requests

// javaSettingsOverrides sobrescreve defaults do Eclipse JDT LS que desabilitam
// funcionalidade util quando o cliente (aqui) nao responde workspace/configuration
// com um valor real. Sem isso, jdtls usa seu default interno para cada
// configuracao pedida - e o default de java.symbols.includeSourceMethodDeclarations
// e false, entao workspace/symbol (usado por FindDefinition/FindReferences) nunca
// retorna metodos/campos, so tipos (classes/interfaces/enums).
var javaSettingsOverrides = map[string]any{
	"symbols": map[string]any{
		"includeSourceMethodDeclarations": true,
	},
}

func HandleWorkspaceConfiguration(params json.RawMessage) (any, error) {
	var configParams protocol.ConfigurationParams
	if err := json.Unmarshal(params, &configParams); err != nil || len(configParams.Items) == 0 {
		return []map[string]any{{}}, nil
	}

	result := make([]any, len(configParams.Items))
	for i, item := range configParams.Items {
		switch item.Section {
		case "java":
			result[i] = javaSettingsOverrides
		case "java.symbols.includeSourceMethodDeclarations":
			result[i] = true
		default:
			result[i] = map[string]any{}
		}
	}
	return result, nil
}

func HandleRegisterCapability(params json.RawMessage) (any, error) {
	var registerParams protocol.RegistrationParams
	if err := json.Unmarshal(params, &registerParams); err != nil {
		lspLogger.Error("Error unmarshaling registration params: %v", err)
		return nil, err
	}

	for _, reg := range registerParams.Registrations {
		lspLogger.Info("Registration received for method: %s, id: %s", reg.Method, reg.ID)

		// Special handling for file watcher registrations
		if reg.Method == "workspace/didChangeWatchedFiles" {
			// Parse the options into the appropriate type
			var opts protocol.DidChangeWatchedFilesRegistrationOptions
			optJson, err := json.Marshal(reg.RegisterOptions)
			if err != nil {
				lspLogger.Error("Error marshaling registration options: %v", err)
				continue
			}

			err = json.Unmarshal(optJson, &opts)
			if err != nil {
				lspLogger.Error("Error unmarshaling registration options: %v", err)
				continue
			}

			// Notify file watchers
			if fileWatchHandler != nil {
				fileWatchHandler(reg.ID, opts.Watchers)
			}
		}
	}

	return nil, nil
}

func HandleApplyEdit(params json.RawMessage) (any, error) {
	var workspaceEdit protocol.ApplyWorkspaceEditParams
	if err := json.Unmarshal(params, &workspaceEdit); err != nil {
		return protocol.ApplyWorkspaceEditResult{Applied: false}, err
	}

	// Apply the edits
	err := utilities.ApplyWorkspaceEdit(workspaceEdit.Edit)
	if err != nil {
		lspLogger.Error("Error applying workspace edit: %v", err)
		return protocol.ApplyWorkspaceEditResult{
			Applied:       false,
			FailureReason: workspaceEditFailure(err),
		}, nil
	}

	return protocol.ApplyWorkspaceEditResult{
		Applied: true,
	}, nil
}

func workspaceEditFailure(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// Notifications

// HandleServerMessage processes window/showMessage notifications from the server
func HandleServerMessage(params json.RawMessage) {
	var msg protocol.ShowMessageParams
	if err := json.Unmarshal(params, &msg); err != nil {
		lspLogger.Error("Error unmarshaling server message: %v", err)
		return
	}

	// Log the message with appropriate level
	switch msg.Type {
	case protocol.Error:
		lspLogger.Error("Server error: %s", msg.Message)
	case protocol.Warning:
		lspLogger.Warn("Server warning: %s", msg.Message)
	case protocol.Info:
		lspLogger.Info("Server info: %s", msg.Message)
	default:
		lspLogger.Debug("Server message: %s", msg.Message)
	}
}

// HandleDiagnostics processes textDocument/publishDiagnostics notifications
func HandleDiagnostics(client *Client, params json.RawMessage) {
	var diagParams protocol.PublishDiagnosticsParams
	if err := json.Unmarshal(params, &diagParams); err != nil {
		lspLogger.Error("Error unmarshaling diagnostic params: %v", err)
		return
	}

	// Save diagnostics in client
	client.diagnosticsMu.Lock()
	client.diagnostics[diagParams.URI] = diagParams.Diagnostics
	client.diagnosticsMu.Unlock()

	lspLogger.Info("Received diagnostics for %s: %d items", diagParams.URI, len(diagParams.Diagnostics))
}
