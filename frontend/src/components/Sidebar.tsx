import { useTranslation } from 'react-i18next';
import { ScrollText, Settings } from 'lucide-react';
import {
  FaDiscord,
  FaInstagram,
  FaXTwitter,
  FaTiktok,
  FaUserAstronaut,
  FaGoogleDrive,
} from 'react-icons/fa6';
import appIcon from '../assets/images/appicon.png';
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';
import type { Page } from '../App';
import './Sidebar.css';
import Button from './hud/Button';
import Tooltip from './hud/Tooltip';

interface SidebarProps {
  activePage: Page;
  onNavigate: (page: Page) => void;
}

const SOCIAL_LINKS = {
  discord: 'https://discord.com/invite/6UTHaANCEz',
  instagram: 'https://www.instagram.com/lethalmon.flo/',
  x: 'https://x.com/lethalmon?s=21',
  tiktok: 'https://www.tiktok.com/@lethalmon.flo',
  googledrive:
    'https://drive.google.com/drive/folders/1JZcs6YW8kcmRPd0kSVlMmQwkMFGcOmLk?dmr=1&ec=wgc-drive-%5Bmodule%5D-goto',
};

function Sidebar({ activePage, onNavigate }: SidebarProps) {
  const { t } = useTranslation();

  return (
    <nav className="sidebar">
      <img className="sidebar-logo" src={appIcon} alt="Lethalmon Launcher" />

      <div className="sidebar-nav">
        <span className="sidebar-icon-wrap">
          <Tooltip text={t('sidebar.home')} placement="right">
            <Button
              variant="select"
              selected={activePage === 'home'}
              icon={<FaUserAstronaut size={22} />}
              iconOnly
              aria-label={t('sidebar.home')}
              onClick={() => onNavigate('home')}
            />
          </Tooltip>
        </span>
        <span className="sidebar-icon-wrap">
          <Tooltip text={t('sidebar.changelog')} placement="right">
            <Button
              variant="select"
              selected={activePage === 'changelog'}
              icon={<ScrollText size={22} />}
              iconOnly
              aria-label={t('sidebar.changelog')}
              onClick={() => onNavigate('changelog')}
            />
          </Tooltip>
        </span>
        <span className="sidebar-icon-wrap">
          <Tooltip text={t('sidebar.settings')} placement="right">
            <Button
              variant="select"
              selected={activePage === 'settings'}
              icon={<Settings size={22} />}
              iconOnly
              aria-label={t('sidebar.settings')}
              onClick={() => onNavigate('settings')}
            />
          </Tooltip>
        </span>
      </div>

      <div className="sidebar-divider" />

      <div className="sidebar-socials">
        <span className="sidebar-social-wrap">
          <Tooltip text={t('sidebar.discord')} placement="right">
            <Button
              icon={<FaDiscord size={22} />}
              iconOnly
              aria-label={t('sidebar.discord')}
              onClick={() => BrowserOpenURL(SOCIAL_LINKS.discord)}
            />
          </Tooltip>
        </span>
        <span className="sidebar-social-wrap">
          <Tooltip text={t('sidebar.instagram')} placement="right">
            <Button
              icon={<FaInstagram size={22} />}
              iconOnly
              aria-label={t('sidebar.instagram')}
              onClick={() => BrowserOpenURL(SOCIAL_LINKS.instagram)}
            />
          </Tooltip>
        </span>
        <span className="sidebar-social-wrap">
          <Tooltip text={t('sidebar.x')} placement="right">
            <Button
              icon={<FaXTwitter size={22} />}
              iconOnly
              aria-label={t('sidebar.x')}
              onClick={() => BrowserOpenURL(SOCIAL_LINKS.x)}
            />
          </Tooltip>
        </span>
        <span className="sidebar-social-wrap">
          <Tooltip text={t('sidebar.tiktok')} placement="right">
            <Button
              icon={<FaTiktok size={22} />}
              iconOnly
              aria-label={t('sidebar.tiktok')}
              onClick={() => BrowserOpenURL(SOCIAL_LINKS.tiktok)}
            />
          </Tooltip>
        </span>
        <span className="sidebar-social-wrap">
          <Tooltip text={t('sidebar.googledrive')} placement="right">
            <Button
              icon={<FaGoogleDrive size={22} />}
              iconOnly
              aria-label={t('sidebar.googledrive')}
              onClick={() => BrowserOpenURL(SOCIAL_LINKS.googledrive)}
            />
          </Tooltip>
        </span>
      </div>
    </nav>
  );
}

export default Sidebar;
