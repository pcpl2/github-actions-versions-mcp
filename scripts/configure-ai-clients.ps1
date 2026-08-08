<#
.SYNOPSIS
    Registers the gha-mcp MCP server with the AI tools installed on this machine.

.DESCRIPTION
    Adds a "github-actions" MCP server entry to Claude Desktop, Cursor and VS Code,
    and to Claude Code via its CLI. Existing servers are preserved: the JSON is
    read, one key is added or replaced, and the file is written back. Every file is
    backed up to <name>.bak first.

    Only tools that are actually installed are touched — a client is detected by
    the presence of its configuration directory.

.PARAMETER BinaryPath
    Full path to gha-mcp.exe. Defaults to the copy sitting next to this script,
    then to whatever is on PATH.

.PARAMETER GithubToken
    Optional GitHub token, stored in the server's env block. Raises the GitHub API
    rate limit from 60 to 5000 requests/hour.

.PARAMETER ServerName
    Name the server is registered under. Defaults to "github-actions".

.PARAMETER Uninstall
    Remove the entry instead of adding it.

.EXAMPLE
    .\configure-ai-clients.ps1

.EXAMPLE
    .\configure-ai-clients.ps1 -GithubToken ghp_xxx

.EXAMPLE
    .\configure-ai-clients.ps1 -Uninstall
#>
[CmdletBinding()]
param(
    [string]$BinaryPath,
    [string]$GithubToken,
    [string]$ServerName = 'github-actions',
    [switch]$Uninstall
)

$ErrorActionPreference = 'Stop'

function Write-Step { param([string]$Message) Write-Host "  $Message" }
function Write-Ok { param([string]$Message) Write-Host "  [ok] $Message" -ForegroundColor Green }
function Write-Skip { param([string]$Message) Write-Host "  [--] $Message" -ForegroundColor DarkGray }
function Write-Warn { param([string]$Message) Write-Host "  [!!] $Message" -ForegroundColor Yellow }

function Resolve-BinaryPath {
    param([string]$Explicit)

    if ($Explicit) {
        if (-not (Test-Path -LiteralPath $Explicit)) {
            throw "gha-mcp not found at '$Explicit'."
        }
        return (Resolve-Path -LiteralPath $Explicit).Path
    }

    $sibling = Join-Path $PSScriptRoot 'gha-mcp.exe'
    if (Test-Path -LiteralPath $sibling) {
        return (Resolve-Path -LiteralPath $sibling).Path
    }

    $onPath = Get-Command 'gha-mcp' -ErrorAction SilentlyContinue
    if ($onPath) { return $onPath.Source }

    throw "Could not find gha-mcp.exe. Pass -BinaryPath with its full path."
}

function Get-JsonObject {
    param([string]$Path)

    if (-not (Test-Path -LiteralPath $Path)) { return New-Object PSObject }

    $raw = Get-Content -LiteralPath $Path -Raw -Encoding UTF8
    if ([string]::IsNullOrWhiteSpace($raw)) { return New-Object PSObject }

    # Windows PowerShell cannot parse JSON with comments; VS Code allows them.
    return $raw | ConvertFrom-Json
}

# Windows PowerShell's ConvertTo-Json aligns values to columns, which looks
# alien in a config file people edit by hand. Re-indent it two-space style.
# Operates on whitespace outside strings only; the caller falls back to the
# original text if the result somehow fails to parse.
function Format-Json {
    param([string]$Json)

    $out = New-Object System.Text.StringBuilder
    $indent = 0
    $inString = $false
    $escaped = $false
    $pad = { param($n) ' ' * (2 * $n) }

    foreach ($ch in $Json.ToCharArray()) {
        if ($escaped) { [void]$out.Append($ch); $escaped = $false; continue }
        if ($ch -eq '\') { [void]$out.Append($ch); $escaped = $true; continue }
        if ($ch -eq '"') { $inString = -not $inString; [void]$out.Append($ch); continue }
        if ($inString) { [void]$out.Append($ch); continue }

        switch ($ch) {
            '{' { $indent++; [void]$out.Append($ch).Append("`n").Append((& $pad $indent)) }
            '[' { $indent++; [void]$out.Append($ch).Append("`n").Append((& $pad $indent)) }
            '}' { $indent--; [void]$out.Append("`n").Append((& $pad $indent)).Append($ch) }
            ']' { $indent--; [void]$out.Append("`n").Append((& $pad $indent)).Append($ch) }
            ',' { [void]$out.Append($ch).Append("`n").Append((& $pad $indent)) }
            ':' { [void]$out.Append($ch).Append(' ') }
            default { if ($ch -notmatch '\s') { [void]$out.Append($ch) } }
        }
    }

    # Collapse the empty-container case: "{\n  \n}" -> "{}"
    return ($out.ToString() -replace '(?m)([\{\[])\s*\r?\n\s*([\}\]])', '$1$2')
}

function Save-JsonObject {
    param([string]$Path, $Object)

    $dir = Split-Path -Parent $Path
    if ($dir -and -not (Test-Path -LiteralPath $dir)) {
        New-Item -ItemType Directory -Path $dir -Force | Out-Null
    }
    if (Test-Path -LiteralPath $Path) {
        Copy-Item -LiteralPath $Path -Destination "$Path.bak" -Force
    }

    $json = $Object | ConvertTo-Json -Depth 20 -Compress
    $pretty = Format-Json -Json $json
    try {
        $null = $pretty | ConvertFrom-Json
    }
    catch {
        # Never trade valid JSON for pretty JSON.
        $pretty = $Object | ConvertTo-Json -Depth 20
    }

    # UTF8 without BOM — a BOM breaks some JSON parsers.
    [System.IO.File]::WriteAllText($Path, $pretty + "`n", (New-Object System.Text.UTF8Encoding($false)))
}

function Set-JsonProperty {
    param($Object, [string]$Name, $Value)

    if ($Object.PSObject.Properties.Name -contains $Name) {
        $Object.$Name = $Value
    }
    else {
        $Object | Add-Member -MemberType NoteProperty -Name $Name -Value $Value
    }
}

function New-ServerEntry {
    param([string]$Command, [string]$Token, [switch]$WithType)

    $entry = New-Object PSObject
    if ($WithType) { Set-JsonProperty $entry 'type' 'stdio' }
    Set-JsonProperty $entry 'command' $Command
    if ($Token) {
        $env = New-Object PSObject
        Set-JsonProperty $env 'GITHUB_TOKEN' $Token
        Set-JsonProperty $entry 'env' $env
    }
    return $entry
}

function Update-ClientConfig {
    param(
        [string]$Label,
        [string]$ConfigPath,
        [string]$RootKey,
        [string]$DetectPath,
        [switch]$WithType
    )

    if (-not (Test-Path -LiteralPath $DetectPath)) {
        Write-Skip "$Label not installed"
        return $false
    }

    try {
        $config = Get-JsonObject -Path $ConfigPath
    }
    catch {
        Write-Warn "$Label - could not parse $ConfigPath (comments in the file?). Configure it by hand."
        return $false
    }

    if ($config.PSObject.Properties.Name -notcontains $RootKey) {
        Set-JsonProperty $config $RootKey (New-Object PSObject)
    }
    $servers = $config.$RootKey

    if ($Uninstall) {
        if ($servers.PSObject.Properties.Name -contains $ServerName) {
            $servers.PSObject.Properties.Remove($ServerName)
            Save-JsonObject -Path $ConfigPath -Object $config
            Write-Ok "$Label - removed '$ServerName'"
            return $true
        }
        Write-Skip "$Label - nothing to remove"
        return $false
    }

    Set-JsonProperty $servers $ServerName (New-ServerEntry -Command $script:binary -Token $GithubToken -WithType:$WithType)
    Save-JsonObject -Path $ConfigPath -Object $config
    Write-Ok "$Label - configured ($ConfigPath)"
    return $true
}

function Update-ClaudeCode {
    $claude = Get-Command 'claude' -ErrorAction SilentlyContinue
    if (-not $claude) {
        Write-Skip 'Claude Code not installed'
        return $false
    }

    if ($Uninstall) {
        & $claude.Source mcp remove $ServerName 2>&1 | Out-Null
        Write-Ok "Claude Code - removed '$ServerName'"
        return $true
    }

    # `mcp add` fails when the name is taken, so drop any previous entry first.
    & $claude.Source mcp remove $ServerName 2>&1 | Out-Null

    $arguments = @('mcp', 'add', $ServerName)
    if ($GithubToken) { $arguments += @('--env', "GITHUB_TOKEN=$GithubToken") }
    $arguments += @('--', $script:binary)

    & $claude.Source @arguments 2>&1 | Out-Null
    if ($LASTEXITCODE -ne 0) {
        Write-Warn "Claude Code - 'claude mcp add' exited with $LASTEXITCODE"
        return $false
    }
    Write-Ok 'Claude Code - configured'
    return $true
}

# ---- main ----

if ($Uninstall) {
    Write-Host "`nRemoving the '$ServerName' MCP server from your AI tools..." -ForegroundColor Cyan
    $script:binary = ''
}
else {
    $script:binary = Resolve-BinaryPath -Explicit $BinaryPath
    Write-Host "`nRegistering gha-mcp with your AI tools..." -ForegroundColor Cyan
    Write-Step "binary: $script:binary"
    if (-not $GithubToken) {
        Write-Step 'no token given - the server will use the anonymous 60 req/h limit'
    }
}

$changed = 0

if (Update-ClientConfig -Label 'Claude Desktop' `
        -ConfigPath (Join-Path $env:APPDATA 'Claude\claude_desktop_config.json') `
        -DetectPath (Join-Path $env:APPDATA 'Claude') `
        -RootKey 'mcpServers') { $changed++ }

if (Update-ClientConfig -Label 'Cursor' `
        -ConfigPath (Join-Path $env:USERPROFILE '.cursor\mcp.json') `
        -DetectPath (Join-Path $env:USERPROFILE '.cursor') `
        -RootKey 'mcpServers') { $changed++ }

if (Update-ClientConfig -Label 'VS Code' `
        -ConfigPath (Join-Path $env:APPDATA 'Code\User\mcp.json') `
        -DetectPath (Join-Path $env:APPDATA 'Code\User') `
        -RootKey 'servers' -WithType) { $changed++ }

if (Update-ClaudeCode) { $changed++ }

Write-Host ''
if ($changed -eq 0) {
    Write-Warn 'No supported AI tools were found. Nothing was changed.'
}
else {
    Write-Host "Done - $changed tool(s) updated. Restart them to pick up the change." -ForegroundColor Cyan
}
