import teamsData from '../assets/images/team_pictures/teams.json';

const images = import.meta.glob('../assets/images/team_pictures/*.{png,webp}', {
  eager: true,
  query: '?url',
  import: 'default',
}) as Record<string, string>;

export type TeamGroup = 'team' | 'contributor';

export interface TeamMember {
  id: string;
  name: string;
  image: string;
  rolesId: string[];
  group: TeamGroup;
  socials: Record<string, string>;
}

function resolveImage(filename: string): string {
  if (!filename) return '';
  const entry = Object.entries(images).find(([path]) => path.endsWith(`/${filename}`));
  return entry?.[1] ?? '';
}

export function getTeamMembers(): TeamMember[] {
  return (teamsData.teams as Array<{
    id: string;
    name: string;
    image: string;
    roles_id: string[];
    group: TeamGroup;
    socials: Record<string, string>;
  }>).map((member) => ({
    id: member.id,
    name: member.name,
    image: resolveImage(member.image),
    rolesId: member.roles_id,
    group: member.group,
    socials: member.socials,
  }));
}
