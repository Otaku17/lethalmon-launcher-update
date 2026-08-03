function stripMarkdown(text: string): string {
  return text
    .replace(/\*\*(.*?)\*\*/g, '$1')
    .replace(/\*(.*?)\*/g, '$1')
    .replace(/`([^`]*)`/g, '$1')
    .replace(/\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replace(/^[-*]\s+/, '')
    .trim();
}

function bodyLines(body: string): string[] {
  return body.split('\n').map((line) => line.trim());
}

/** Extracts a human title: the release name, or the first markdown heading
 * found in the body, or the tag name as a last resort. */
export function extractTitle(release: {
  name: string;
  tag_name: string;
  body: string;
}): string {
  if (release.name) return release.name;

  const firstHeading = bodyLines(release.body || '').find((line) =>
    /^#{1,6}\s/.test(line),
  );
  if (firstHeading) {
    return firstHeading.replace(/^#{1,6}\s*/, '');
  }

  return release.tag_name;
}

/** Extracts a short plain-text preview from a release's markdown body,
 * skipping the leading heading line. */
export function extractExcerpt(body: string, maxLength = 160): string {
  if (!body) return '';

  const lines = bodyLines(body).filter(
    (line) => line !== '' && !/^#{1,6}\s/.test(line),
  );

  let text = stripMarkdown(lines.join(' ')).replace(/\s+/g, ' ');

  if (text.length > maxLength) {
    text = `${text.slice(0, maxLength).trim()}…`;
  }

  return text;
}
