const { app, BrowserWindow } = require('electron');
const path = require('path');
const fs = require('fs');

// Load .env from the correct location (works in both dev and packaged app)
const envPath = path.join(__dirname, '.env');
if (fs.existsSync(envPath)) {
  require('dotenv').config({ path: envPath });
} else {
  // Fallback: try to load from resources directory (packaged app)
  const resourcesPath = process.resourcesPath || path.join(__dirname, '..');
  const envPathInResources = path.join(resourcesPath, '.env');
  if (fs.existsSync(envPathInResources)) {
    require('dotenv').config({ path: envPathInResources });
  }
}

function createWindow() {
  const win = new BrowserWindow({
    width: parseInt(process.env.ELECTRON_WINDOW_WIDTH || '1100', 10),
    height: parseInt(process.env.ELECTRON_WINDOW_HEIGHT || '740', 10),
    minWidth: parseInt(process.env.ELECTRON_WINDOW_MIN_WIDTH || '840', 10),
    minHeight: parseInt(process.env.ELECTRON_WINDOW_MIN_HEIGHT || '540', 10),
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
    // Development mode: load from dev server (ELECTRON_DEV_URL from .env, or Vite dev port)
    const devUrl =
      process.env.ELECTRON_DEV_URL ||
      `http://localhost:${process.env.VITE_PORT || '5173'}`;
    win.loadURL(devUrl);
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
