#define MyAppName "PacketMind"
#define MyAppVersion GetEnv("PACKETMIND_VERSION")
#define MyAppExeName "packetmind.exe"
#define MyArch GetEnv("PACKETMIND_ARCH")

[Setup]
AppId={{9F2C9E7F-39A1-4A3C-A0C1-9A6C5B21D4B6}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher=PacketMind
DefaultDirName={localappdata}\PacketMind
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
OutputDir=..\..\dist
OutputBaseFilename=PacketMindInstaller_{#MyAppVersion}_windows_{#MyArch}
Compression=lzma
SolidCompression=yes
PrivilegesRequired=lowest
WizardStyle=modern
ArchitecturesInstallIn64BitMode=x64compatible

[Files]
Source: "..\..\build\bin\packetmind.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\configs\packetmind.json"; DestDir: "{app}\configs"; Flags: ignoreversion
Source: "..\..\configs\models.json"; DestDir: "{app}\configs"; Flags: ignoreversion

[Tasks]
Name: "desktopicon"; Description: "Create a desktop shortcut"; GroupDescription: "Additional icons:"; Flags: unchecked

[Icons]
Name: "{group}\PacketMind"; Filename: "{app}\{#MyAppExeName}"; WorkingDir: "{app}"
Name: "{commondesktop}\PacketMind"; Filename: "{app}\{#MyAppExeName}"; WorkingDir: "{app}"; Tasks: desktopicon

[Run]
Filename: "{app}\{#MyAppExeName}"; WorkingDir: "{app}"; Description: "Launch PacketMind"; Flags: nowait postinstall skipifsilent
