; Inno Setup script for the gha-mcp Windows installer.
;
; Built by .github/workflows/release.yml, which unpacks the release zip into a
; staging folder and calls:
;
;   ISCC.exe /DAppVersion=1.1.0 /DArch=amd64 /DStageDir=<path> /DOutputDir=<path> gha-mcp.iss
;
; Installs per-user, so it never prompts for elevation.

#ifndef AppVersion
  #define AppVersion "0.0.0"
#endif
#ifndef Arch
  #define Arch "amd64"
#endif
#ifndef StageDir
  #define StageDir "."
#endif
#ifndef OutputDir
  #define OutputDir "."
#endif

#define AppName "gha-mcp"
#define AppPublisher "Patryk Ławicki"
#define AppURL "https://github.com/pcpl2/github-actions-versions-mcp"

[Setup]
; Keep this GUID stable — it is how Windows recognises an upgrade.
AppId={{8F3A7C21-5D64-4B9E-9A17-2C6E0B8D4F13}
AppName={#AppName}
AppVersion={#AppVersion}
AppVerName={#AppName} {#AppVersion}
AppPublisher={#AppPublisher}
AppPublisherURL={#AppURL}
AppSupportURL={#AppURL}/issues
AppUpdatesURL={#AppURL}/releases
DefaultDirName={autopf}\{#AppName}
DefaultGroupName={#AppName}
DisableProgramGroupPage=yes
LicenseFile={#StageDir}\LICENSE
InfoAfterFile={#StageDir}\INSTALL-NOTES.txt
OutputDir={#OutputDir}
OutputBaseFilename={#AppName}_{#AppVersion}_windows_{#Arch}_setup
SetupIconFile=
Compression=lzma2/max
SolidCompression=yes
WizardStyle=modern
; Per-user install: no UAC prompt, PATH changes land in HKCU.
PrivilegesRequired=lowest
ChangesEnvironment=yes
UninstallDisplayName={#AppName} {#AppVersion}
UninstallDisplayIcon={app}\gha-mcp.exe
#if Arch == "arm64"
ArchitecturesAllowed=arm64
#else
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
#endif

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "addtopath"; Description: "Add gha-mcp to my PATH (recommended)"; GroupDescription: "Set up:"
Name: "configureai"; Description: "Configure my AI tools automatically (Claude Desktop, Claude Code, Cursor, VS Code)"; GroupDescription: "Set up:"

[Files]
Source: "{#StageDir}\gha-mcp.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#StageDir}\configure-ai-clients.ps1"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#StageDir}\README.md"; DestDir: "{app}"; Flags: ignoreversion isreadme
Source: "{#StageDir}\CHANGELOG.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#StageDir}\LICENSE"; DestDir: "{app}"; Flags: ignoreversion

[Registry]
Root: HKCU; Subkey: "Environment"; ValueType: expandsz; ValueName: "Path"; \
    ValueData: "{olddata};{app}"; Tasks: addtopath; Check: NeedsAddPath(ExpandConstant('{app}'))

[Run]
Filename: "{sys}\WindowsPowerShell\v1.0\powershell.exe"; \
    Parameters: "-NoProfile -ExecutionPolicy Bypass -File ""{app}\configure-ai-clients.ps1"" -BinaryPath ""{app}\gha-mcp.exe"""; \
    StatusMsg: "Configuring your AI tools..."; \
    Flags: runhidden waituntilterminated; Tasks: configureai
Filename: "{app}\README.md"; Description: "Open the README"; \
    Flags: postinstall shellexec skipifsilent nowait unchecked

[UninstallRun]
Filename: "{sys}\WindowsPowerShell\v1.0\powershell.exe"; \
    Parameters: "-NoProfile -ExecutionPolicy Bypass -File ""{app}\configure-ai-clients.ps1"" -Uninstall"; \
    RunOnceId: "RemoveMcpEntries"; Flags: runhidden waituntilterminated

[Code]
{ True when the install directory is not already on the user's PATH. }
function NeedsAddPath(Param: string): boolean;
var
  OrigPath: string;
begin
  if not RegQueryStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', OrigPath) then
  begin
    Result := True;
    exit;
  end;
  Result := Pos(';' + Uppercase(Param) + ';', ';' + Uppercase(OrigPath) + ';') = 0;
end;

{ Take the install directory back out of PATH on uninstall. }
procedure RemoveFromPath(Dir: string);
var
  OrigPath: string;
  NewPath: string;
  P: Integer;
begin
  if not RegQueryStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', OrigPath) then
    exit;

  NewPath := ';' + OrigPath + ';';
  P := Pos(';' + Uppercase(Dir) + ';', Uppercase(NewPath));
  if P = 0 then
    exit;

  Delete(NewPath, P, Length(Dir) + 1);
  { Strip the sentinel semicolons we added. }
  if (Length(NewPath) > 0) and (NewPath[1] = ';') then
    Delete(NewPath, 1, 1);
  if (Length(NewPath) > 0) and (NewPath[Length(NewPath)] = ';') then
    Delete(NewPath, Length(NewPath), 1);

  RegWriteExpandStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', NewPath);
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usPostUninstall then
    RemoveFromPath(ExpandConstant('{app}'));
end;
