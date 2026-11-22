Postwoman portable for Windows
==============================

This folder contains a self-contained Postwoman build. No installer or
administrator privileges are required.

Quick start
-----------
1. Extract the whole folder to a writable location (for example a USB drive
   or a folder on your desktop).
2. Run Postwoman.exe.
3. Your collections, environments, history and settings are stored in the
   "data" folder next to Postwoman.exe. The included portable.txt marker is
   what enables this portable data mode. Keep both files together.
4. To move your setup, copy the entire folder - including "data" - to the new
   location.

Requirements
------------
- Windows 10 or Windows 11, 64-bit.
- Microsoft Edge WebView2 Runtime. The Postwoman executable embeds the
  Microsoft WebView2 bootstrapper and offers to install the runtime if it is
  missing. Windows 10/11 normally already has it.

Troubleshooting
---------------
- If Windows SmartScreen warns about an unsigned download, choose
  "More info" -> "Run anyway" only if you downloaded the archive from the
  official Postwoman GitHub release and verified SHA256SUMS.txt.
- If the window does not open, install the WebView2 Runtime from
  https://developer.microsoft.com/microsoft-edge/webview2/ and try again.
- Do not run Postwoman from a read-only folder or a read-only USB drive; the
  application needs to write its database and WebView2 cache.
- To reset the portable application, close Postwoman and delete the "data"
  folder. This permanently removes your local Postwoman data.

License
-------
MIT. See LICENSE in this folder.