const { app, BrowserWindow } = require('electron');
const path = require('path');
const fs = require('fs');

function createWindow() {
  const win = new BrowserWindow({
    width: 1100,
    height: 740,
    minWidth: 840,
    minHeight: 540,
    backgroundColor: '#171717',
    autoHideMenuBar: true,
    title: 'UnderMine Messenger',
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
    },
  });

  const distHtml = path.join(__dirname, 'dist', 'index.html');

  if (fs.existsSync(distHtml)) {
    // Production .exe: load bundled offline frontend
    win.loadFile(distHtml);
  } else {
    // Development mode: load from dev server
    win.loadURL('http://localhost:3000');
  }
}

app.whenReady().then(() => {
  createWindow();

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow();
  });
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') app.quit();
});
