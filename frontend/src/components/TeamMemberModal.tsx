import { ComponentType } from 'react';
import { useTranslation } from 'react-i18next';
import {
  FaUserAstronaut,
  FaGithub,
  FaDiscord,
  FaXTwitter,
  FaInstagram,
  FaTiktok,
  FaGoogleDrive,
  FaGlobe,
} from 'react-icons/fa6';
import Modal from './hud/Modal';
import type { TeamMember } from '../lib/team';
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';
import './TeamMemberModal.css';

const SOCIAL_ICONS: Record<string, ComponentType<{ size?: number }>> = {
  github: FaGithub,
  discord: FaDiscord,
  x: FaXTwitter,
  twitter: FaXTwitter,
  instagram: FaInstagram,
  tiktok: FaTiktok,
  googledrive: FaGoogleDrive,
  website: FaGlobe,
};

export interface TeamMemberModalProps {
  member: TeamMember | null;
  onClose: () => void;
}

/**
 * Detailed profile for a team member (avatar, roles, links), opened by
 * clicking a card in the team grid.
 *
 * <TeamMemberModal member={selected} onClose={() => setSelected(null)} />
 */
export default function TeamMemberModal({ member, onClose }: TeamMemberModalProps) {
  const { t } = useTranslation();
  if (!member) return null;

  const socials = Object.entries(member.socials).filter(([, url]) => url);

  return (
    <Modal
      variant="close"
      open
      eyebrow="// FICHE_MEMBRE"
      title={member.name}
      onClose={onClose}
    >
      <div className="team-member-modal">
        <div className="team-member-modal__identity">
          {member.image ? (
            <img className="team-member-modal__avatar" src={member.image} alt={member.name} />
          ) : (
            <div className="team-member-modal__avatar team-member-modal__avatar--placeholder">
              <FaUserAstronaut size={26} />
            </div>
          )}
          <div className="team-member-modal__roles">
            {member.rolesId.map((roleId) => t(`settings.about.roles.${roleId}`)).join(' · ')}
          </div>
        </div>

        {socials.length > 0 && (
          <div className="team-member-modal__socials">
            {socials.map(([key, url]) => {
              const Icon = SOCIAL_ICONS[key] ?? FaGlobe;
              return (
                <button
                  key={key}
                  type="button"
                  className="team-member-modal__social"
                  aria-label={key}
                  onClick={() => BrowserOpenURL(url)}
                >
                  <Icon size={16} />
                </button>
              );
            })}
          </div>
        )}
      </div>
    </Modal>
  );
}
