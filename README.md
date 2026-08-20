# MCP Language Server

[![Go Tests](https://github.com/isaacphi/mcp-language-server/actions/workflows/go.yml/badge.svg)](https://github.com/isaacphi/mcp-language-server/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/isaacphi/mcp-language-server)](https://goreportcard.com/report/github.com/isaacphi/mcp-language-server)
[![GoDoc](https://pkg.go.dev/badge/github.com/isaacphi/mcp-language-server)](https://pkg.go.dev/github.com/isaacphi/mcp-language-server)
[![Go Version](https://img.shields.io/github/go-mod/go-version/isaacphi/mcp-language-server)](https://github.com/isaacphi/mcp-language-server/blob/main/go.mod)

This is an [MCP](https://modelcontextprotocol.io/introduction) server that runs and exposes a [language server](https://microsoft.github.io/language-server-protocol/) to LLMs. Not a language server for MCP, whatever that would be.

## Demo

`mcp-language-server` helps MCP enabled clients navigate codebases more easily by giving them access semantic tools like get definition, references, rename, and diagnostics.

![Demo](demo.gif)

## Setup

1. **Install Go**: Follow instructions at <https://golang.org/doc/install>
2. **Install or update this server**: `go install github.com/isaacphi/mcp-language-server@latest`
3. **Install a language server**: _follow one of the guides below_
4. **Configure your MCP client**: _follow one of the guides below_

<details>
  <summary>Go (gopls)</summary>
  <div>
    <p><strong>Install gopls</strong>: <code>go install golang.org/x/tools/gopls@latest</code></p>
    <p><strong>Configure your MCP client</strong>: This will be different but similar for each client. For Claude Desktop, add the following to <code>~/Library/Application\ Support/Claude/claude_desktop_config.json</code></p>

<pre>
{
  "mcpServers": {
    "language-server": {
      "command": "mcp-language-server",
      "args": ["--workspace", "/Users/you/dev/yourproject/", "--lsp", "gopls"],
      "env": {
        "PATH": "/opt/homebrew/bin:/Users/you/go/bin",
        "GOPATH": "/users/you/go",
        "GOCACHE": "/users/you/Library/Caches/go-build",
        "GOMODCACHE": "/Users/you/go/pkg/mod"
      }
    }
  }
}
</pre>

<p><strong>Note</strong>: Not all clients will need these environment variables. For Claude Desktop you will need to update the environment variables above based on your machine and username:</p>
<ul>
  <li><code>PATH</code> needs to contain the path to <code>go</code> and to <code>gopls</code>. Get this with <code>echo $(which go):$(which gopls)</code></li>
  <li><code>GOPATH</code>, <code>GOCACHE</code>, and <code>GOMODCACHE</code> may be different on your machine. These are the defaults.</li>
</ul>

  </div>
</details>
<details>
  <summary>Rust (rust-analyzer)</summary>
  <div>
    <p><strong>Install rust-analyzer</strong>: <code>rustup component add rust-analyzer</code></p>
    <p><strong>Configure your MCP client</strong>: This will be different but similar for each client. For Claude Desktop, add the following to <code>~/Library/Application\ Support/Claude/claude_desktop_config.json</code></p>

<pre>
{
  "mcpServers": {
    "language-server": {
      "command": "mcp-language-server",
      "args": [
        "--workspace",
        "/Users/you/dev/yourproject/",
        "--lsp",
        "rust-analyzer"
      ]
    }
  }
}
</pre>
  </div>
</details>
<details>
  <summary>Python (pyright)</summary>
  <div>
    <p><strong>Install pyright</strong>: <code>npm install -g pyright</code></p>
    <p><strong>Configure your MCP client</strong>: This will be different but similar for each client. For Claude Desktop, add the following to <code>~/Library/Application\ Support/Claude/claude_desktop_config.json</code></p>

<pre>
{
  "mcpServers": {
    "language-server": {
      "command": "mcp-language-server",
      "args": [
        "--workspace",
        "/Users/you/dev/yourproject/",
        "--lsp",
        "pyright-langserver",
        "--",
        "--stdio"
      ]
    }
  }
}
</pre>
  </div>
</details>
<details>
  <summary>Typescript (typescript-language-server)</summary>
  <div>
    <p><strong>Install typescript-language-server</strong>: <code>npm install -g typescript typescript-language-server</code></p>
    <p><strong>Configure your MCP client</strong>: This will be different but similar for each client. For Claude Desktop, add the following to <code>~/Library/Application\ Support/Claude/claude_desktop_config.json</code></p>

<pre>
{
  "mcpServers": {
    "language-server": {
      "command": "mcp-language-server",
      "args": [
        "--workspace",
        "/Users/you/dev/yourproject/",
        "--lsp",
        "typescript-language-server",
        "--",
        "--stdio"
      ]
    }
  }
}
</pre>
  </div>
</details>
<details>
  <summary>C/C++ (clangd)</summary>
  <div>
    <p><strong>Install clangd</strong>: Download prebuilt binaries from the <a href="https://github.com/clangd/clangd/releases">official LLVM releases page</a> or install via your system's package manager (e.g., <code>apt install clangd</code>, <code>brew install clangd</code>).</p>
    <p><strong>Configure your MCP client</strong>: This will be different but similar for each client. For Claude Desktop, add the following to <code>~/Library/Application\\ Support/Claude/claude_desktop_config.json</code></p>

<pre>
{
  "mcpServers": {
    "language-server": {
      "command": "mcp-language-server",
      "args": [
        "--workspace",
        "/Users/you/dev/yourproject/",
        "--lsp",
        "/path/to/your/clangd_binary",
        "--",
        "--compile-commands-dir=/path/to/yourproject/build_or_compile_commands_dir"
      ]
    }
  }
}
</pre>
    <p><strong>Note</strong>:</p>
    <ul>
      <li>Replace <code>/path/to/your/clangd_binary</code> with the actual path to your clangd executable.</li>
      <li><code>--compile-commands-dir</code> should point to the directory containing your <code>compile_commands.json</code> file (e.g., <code>./build</code>, <code>./cmake-build-debug</code>).</li>
      <li>Ensure <code>compile_commands.json</code> is generated for your project for clangd to work effectively.</li>
    </ul>
  </div>
</details>
<details>
  <summary>Other</summary>
  <div>
    <p>I have only tested this repo with the servers above but it should be compatible with many more. Note:</p>
    <ul>
      <li>The language server must communicate over stdio.</li>
      <li>Any aruments after <code>--</code> are sent as arguments to the language server.</li>
      <li>Any env variables are passed on to the language server.</li>
    </ul>
  </div>
</details>

## Tools

- `definition`: Retrieves the complete source code definition of any symbol (function, type, constant, etc.) from your codebase.
- `references`: Locates all usages and references of a symbol throughout the codebase.
- `diagnostics`: Provides diagnostic information for a specific file, including warnings and errors.
- `hover`: Display documentation, type hints, or other hover information for a given location.
- `rename_symbol`: Rename a symbol across a project.
- `edit_file`: Allows making multiple text edits to a file based on line numbers. Provides a more reliable and context-economical way to edit files compared to search and replace based edit tools.

## This fork (beesmart-app/mcp-language-server)

This is a fork of [isaacphi/mcp-language-server](https://github.com/isaacphi/mcp-language-server) with fixes and one new feature that hadn't landed upstream as of `v0.1.1`. Module path and binary name are unchanged, so it's a drop-in replacement.

**Fixes** (each is a separate tag, `v0.1.1-beesmart.N`):

1. **Write mutex**: `WriteMessage` wrote the `Content-Length` header and JSON body to `stdin` in two separate calls with no lock. Under concurrent writers (e.g. a burst of `workspace/didChangeWatchedFiles` racing a tool call response), bytes interleaved on the pipe and the language server lost LSP framing sync, entering a "Missing header Content-Length" error loop that degrades or kills the connection.
2. **`workspace/symbol` for methods**: `HandleWorkspaceConfiguration` answered every config request with `{}`, so `java.symbols.includeSourceMethodDeclarations` (default `false` in Eclipse JDT LS) never got enabled and symbol search only ever returned types. Separately, qualified names (`Type.Method`, the format the tool itself documents) were sent to the server with a literal `.`, which the server's fuzzy matcher doesn't understand and returns empty for — fixed by searching on the method name alone and filtering by container name client-side.
3. **Hover with legacy `MarkedString[]`**: some servers (jdtls included) sometimes reply to `hover` with the legacy LSP array-of-`MarkedString` format instead of modern `MarkupContent`, which the generated type only accepted one of. Fixed with a custom `UnmarshalJSON` that accepts both.
4. **Stale diagnostics after external edits**: `GetDiagnosticsForFile` called `OpenFile`, a no-op if the file was already open in this session, then a flat `time.Sleep(3s)` before reading a diagnostics cache that's only updated asynchronously by the file watcher. If the watcher hadn't caught up to an edit within 3s (easy on a larger project, or edits in quick succession), the tool silently served pre-edit diagnostics with no versioning to catch it. Fixed by forcing a `NotifyChange` unconditionally before waiting, and made the wait configurable via `LSP_DIAGNOSTICS_WAIT_SECONDS` (same pattern as `LSP_CONTEXT_LINES`) since 3s isn't always enough under load.
5. **Daemon mode** (`v0.1.1-beesmart.6`) — see below.

### Daemon mode: sharing one language server across multiple MCP clients

By default (as upstream), this binary serves MCP over stdio: one process per client, 1:1. If several MCP clients (e.g. several editor windows/sessions) point at the same `-workspace`, each spawns its own language server — for a stateful server like jdtls this means duplicate JVMs and, worse, multiple processes writing to the same on-disk language-server workspace/index with no coordination (we hit real index corruption this way).

New flags let one process serve many clients concurrently instead:

- `-listen <unix-socket-path>`: instead of `ServeStdio`, serve MCP over SSE (`server.NewSSEServer`, from `mark3labs/mcp-go`) on a Unix domain socket. All connected clients share the same `*lsp.Client` / language server process — tools are registered once, over that shared client. In this mode the parent-process watchdog is disabled (a daemon must outlive any single client that started it).
- `-idle-timeout <duration>` (default `20m`, daemon mode only): shuts down gracefully (closes files, sends LSP `shutdown`/`exit`) after this long with zero connected clients, using the `OnRegisterSession`/`OnUnregisterSession` hooks to track active sessions.

`cmd/bridge` is a small, protocol-generic companion binary (no language-specific logic) that speaks MCP-over-stdio on one side (so it's still a normal stdio "command" from the client's point of view) and MCP-over-SSE to a daemon's Unix socket on the other, forwarding `ListTools`/`CallTool` 1:1. This lets you keep a stdio-based MCP client config unchanged while multiple client processes share one backend:

```bash
# once, or lazily on first use - start (or reuse) the shared daemon:
mcp-language-server -workspace /path/to/project -listen /run/user/$UID/my-project.sock -idle-timeout 20m -lsp gopls &

# each client's MCP config points at the bridge instead of the daemon directly:
# "command": "mcp-language-server-bridge", "args": ["-socket", "/run/user/1000/my-project.sock"]
```

Bootstrapping (deciding whether a daemon is already up, starting one detached if not, and waiting for the socket to become live) is left to the caller — this repo doesn't ship a supervisor for that. Note the ~108-byte `sun_path` limit on Unix sockets: keep the socket path short (e.g. under `$XDG_RUNTIME_DIR`) rather than nesting it inside a possibly-deep workspace/data directory.

## About

This codebase makes use of edited code from [gopls](https://go.googlesource.com/tools/+/refs/heads/master/gopls/internal/protocol) to handle LSP communication. See ATTRIBUTION for details. Everything here is covered by a permissive BSD style license.

[mcp-go](https://github.com/mark3labs/mcp-go) is used for MCP communication. Thank you for your service.

This is beta software. Please let me know by creating an issue if you run into any problems or have suggestions of any kind.

## Contributing

Please keep PRs small and open Issues first for anything substantial. AI slop O.K. as long as it is tested, passes checks, and doesn't smell too bad.

### Setup

Clone the repo:

```bash
git clone https://github.com/isaacphi/mcp-language-server.git
cd mcp-language-server
```

A [justfile](https://just.systems/man/en/) is included for convenience:

```bash
just -l
Available recipes:
    build    # Build
    check    # Run code audit checks
    fmt      # Format code
    generate # Generate LSP types and methods
    help     # Help
    install  # Install locally
    snapshot # Update snapshot tests
    test     # Run tests
```

Configure your Claude Desktop (or similar) to use the local binary:

```json
{
  "mcpServers": {
    "language-server": {
      "command": "/full/path/to/your/clone/mcp-language-server/mcp-language-server",
      "args": [
        "--workspace",
        "/path/to/workspace",
        "--lsp",
        "language-server-executable"
      ],
      "env": {
        "LOG_LEVEL": "DEBUG"
      }
    }
  }
}
```

Rebuild after making changes.

### Logging

Setting the `LOG_LEVEL` environment variable to DEBUG enables verbose logging to stderr for all components including messages to and from the language server and the language server's logs.

### LSP interaction

- `internal/lsp/methods.go` contains generated code to make calls to the connected language server.
- `internal/protocol/tsprotocol.go` contains generated code for LSP types. I borrowed this from `gopls`'s source code. Thank you for your service.
- LSP allows language servers to return different types for the same methods. Go doesn't like this so there are some ugly workarounds in `internal/protocol/interfaces.go`.

### Local Development and Snapshot Tests

There is a snapshot test suite that makes it a lot easier to try out changes to tools. These run actual language servers on mock workspaces and capture output and logs.

You will need the language servers installed locally to run them. There are tests for go, rust, python, and typescript.

```
integrationtests/
├── tests/        # Tests are in this folder
├── snapshots/    # Snapshots of tool outputs
├── test-output/  # Gitignored folder showing the final state of each workspace and logs after each test run
└── workspaces/   # Mock workspaces that the tools run on
```

To update snapshots, run `UPDATE_SNAPSHOTS=true go test ./integrationtests/...`
