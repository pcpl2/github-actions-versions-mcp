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
; Publisher is the brand shown in Add/Remove Programs — it must match the
; Publisher field of the WinGet manifest, or `winget upgrade` fails to correlate
; the installed program with the package. Copyright stays with the author.
#define AppPublisher "Pcpl2Lab"
#define AppAuthor "Patryk Ławicki"
#define AppURL "https://github.com/pcpl2/github-actions-versions-mcp"
#define RepoRoot "..\.."

; VersionInfoVersion only accepts numbers, so drop any pre-release suffix
; (1.2.3-rc.1 -> 1.2.3) while AppVersion keeps the full string.
#define DashPos Pos("-", AppVersion)
#if DashPos > 0
  #define BaseVersion Copy(AppVersion, 1, DashPos - 1)
#else
  #define BaseVersion AppVersion
#endif

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
Compression=lzma2/max
SolidCompression=yes
WizardStyle=modern

; Branding. Two sizes each so Setup can pick the sharper one per DPI.
SetupIconFile={#RepoRoot}\assets\icon.ico
WizardImageFile=wizard-large-240x459.png,wizard-large-480x918.png
WizardSmallImageFile=wizard-small-147x147.png,wizard-small-294x294.png

; Metadata shown on the setup executable's Properties > Details tab.
VersionInfoVersion={#BaseVersion}
VersionInfoProductVersion={#BaseVersion}
VersionInfoCompany={#AppPublisher}
VersionInfoProductName={#AppName}
VersionInfoProductTextVersion={#AppVersion}
VersionInfoDescription={#AppName} {#AppVersion} Setup
VersionInfoTextVersion={#AppVersion}
VersionInfoCopyright=Copyright (c) 2026 {#AppAuthor}. BSD 2-Clause licence.
VersionInfoOriginalFileName={#AppName}_{#AppVersion}_windows_{#Arch}_setup.exe
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
