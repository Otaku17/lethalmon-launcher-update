import { useEffect, useState } from 'react';
import './App.css';
import Sidebar from './components/Sidebar';
import ParticleBackground from './components/ParticleBackground';
import HomePage from './pages/HomePage';
import ChangelogPage from './pages/ChangelogPage';
import SettingsPage from './pages/SettingsPage';
import WindowControls from './components/hud/WindowControls';
import OnlinePlayers from './components/hud/OnlinePlayers';
import LauncherUpdateButton from './components/hud/LauncherUpdateButton';
import { AppStatusProvider } from './lib/appStatus';

export type Page = 'home' | 'changelog' | 'settings';

function App() {
  const [activePage, setActivePage] = useState<Page>('home');

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'F5') {
        e.preventDefault();
        window.location.reload();
      }
    }
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  return (
    <AppStatusProvider>
      <div id="app">
        <div
          className="drag-region"
          style={{ '--wails-draggable': 'drag' } as React.CSSProperties}
        ></div>
        {/*<CaptionBar label="Lethalmon Launcher" />*/}
        <div className="titlebar-status">
          <LauncherUpdateButton />
          <OnlinePlayers />
          <WindowControls />
        </div>
        <div className="app-layout">
          <Sidebar activePage={activePage} onNavigate={setActivePage} />
          <main className="app-content">
            <ParticleBackground />
            <div className="app-content-inner">
              {activePage === 'home' && <HomePage />}
              {activePage === 'changelog' && <ChangelogPage />}
              {activePage === 'settings' && <SettingsPage />}
            </div>
          </main>
        </div>
      </div>
    </AppStatusProvider>
  );
}

export default App;
