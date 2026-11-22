Unicode true

####
## Postwoman NSIS installer.
##
## The version, company, product and executable names are resolved by
## wails_tools.nsh from wails.json (info.*) at build time. Do not hard-code the
## version here: bump VERSION and run scripts/update-version.mjs instead.
##
## Build with:
##   wails build -platform windows/amd64 -nsis -installscope user
####

!define INFO_PROJECTNAME    "Postwoman"
!define INFO_PRODUCTNAME    "Postwoman"
!define INFO_COMPANYNAME    "Postwoman"
!define PRODUCT_EXECUTABLE  "Postwoman.exe"
!define UNINST_KEY_NAME     "Postwoman"

!include "wails_tools.nsh"

# Version information must consist of four numeric parts.
VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

# HiDPI-aware installer UI.
ManifestDPIAware true

!include "MUI.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_ABORTWARNING
!define MUI_FINISHPAGE_NOAUTOCLOSE
!define MUI_FINISHPAGE_RUN "$INSTDIR\${PRODUCT_EXECUTABLE}"
!define MUI_FINISHPAGE_RUN_TEXT "Launch Postwoman"

!insertmacro MUI_PAGE_WELCOME
# Components page lets the user opt in to the desktop shortcut.
!insertmacro MUI_PAGE_COMPONENTS
# Directory page lets the user choose where Postwoman is installed.
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe"

# `-installscope user` from Wails installs per-user (no administrator prompt)
# under %LOCALAPPDATA%\Programs. Machine installs fall back to Program Files.
!ifdef WAILS_INSTALL_SCOPE
  !if "${WAILS_INSTALL_SCOPE}" == "user"
    InstallDir "$LOCALAPPDATA\Programs\${INFO_PRODUCTNAME}"
  !else
    InstallDir "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
  !endif
!else
  InstallDir "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
!endif

ShowInstDetails show
ShowUninstDetails show

Function .onInit
  !insertmacro wails.checkArchitecture
FunctionEnd

Section "Postwoman (required)" SEC_CORE
  SectionIn RO
  !insertmacro wails.setShellContext
  # Installs the WebView2 Evergreen runtime if it is missing (bootstrapper is
  # embedded in the installer by Wails).
  !insertmacro wails.webview2runtime

  SetOutPath $INSTDIR
  !insertmacro wails.files

  # Remove an older desktop shortcut first; it is recreated only when the
  # optional Desktop shortcut component is selected.
  Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

  # Start Menu shortcut is always created. Existing shortcuts are overwritten
  # during upgrades, so no duplicate entries accumulate.
  CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

  !insertmacro wails.associateFiles
  !insertmacro wails.associateCustomProtocols
  !insertmacro wails.writeUninstaller
SectionEnd

Section /o "Desktop shortcut" SEC_DESKTOP
  !insertmacro wails.setShellContext
  CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
SectionEnd

Section "Uninstall"
  !insertmacro wails.setShellContext

  Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
  Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

  !insertmacro wails.unassociateFiles
  !insertmacro wails.unassociateCustomProtocols
  !insertmacro wails.deleteUninstaller

  # Remove only the installation directory. Application data lives in
  # %APPDATA%\Postwoman (installer build) or the portable data directory and is
  # deliberately preserved across upgrades and uninstalls.
  RMDir /r "$INSTDIR"
SectionEnd