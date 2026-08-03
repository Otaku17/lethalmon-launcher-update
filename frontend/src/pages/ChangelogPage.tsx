import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { getReleases, type ReleaseKind } from '../lib/releasesCache';
import { extractTitle, extractExcerpt } from '../lib/markdownExcerpt';
import { main } from '../../wailsjs/go/models';
import './Pages.css';
import './ChangelogPage.css';
import Modal from '../components/hud/Modal';
import Panel from '../components/hud/Panel';
import { Badge } from '../components/hud/Badge';
import Tabs from '../components/hud/Tabs';
import { useAppStatus } from '../lib/appStatus';

const TABS: ReleaseKind[] = ['game', 'launcher'];

function groupLabelKey(index: number): string {
  if (index === 0) return 'changelog.groupLatest';
  if (index === 1) return 'changelog.groupPrevious';
  return 'changelog.groupEarlier';
}

function ChangelogPage() {
  const { t, i18n } = useTranslation();
  const { online } = useAppStatus();
  const [tab, setTab] = useState<ReleaseKind>('game');
  const [releases, setReleases] = useState<main.Release[] | null>(null);
  const [error, setError] = useState(false);
  const [selected, setSelected] = useState<main.Release | null>(null);

  useEffect(() => {
    if (!online) {
      setReleases(null);
      setError(false);
      return;
    }

    setReleases(null);
    setError(false);
    getReleases(tab)
      .then(setReleases)
      .catch(() => setError(true));
  }, [tab, online]);

  function formatDate(value: string) {
    return new Date(value).toLocaleDateString(i18n.language, {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  }

  return (
    <div className="changelog-page">
      <Panel
        eyebrow="// JOURNAL_DE_BORD"
        title={t('changelog.title')}
        headerExtra={<Badge variant="warn">{t('changelog.disclaimer')}</Badge>}
        className="changelog-panel"
      >
        <p className="changelog-description">
          {t('changelog.description')}
        </p>

        <Tabs
          className="changelog-tabs"
          activeId={tab}
          onChange={(id) => setTab(id as ReleaseKind)}
          tabs={TABS.map((kind) => ({
            id: kind,
            label: t(`changelog.tabs.${kind}`),
            content: (
              <div className="changelog-list">
                {!online && (
                  <p className="page-placeholder">{t('changelog.offline')}</p>
                )}

                {online && error && (
                  <p className="page-placeholder">{t('changelog.error')}</p>
                )}

                {online && !error && !releases && (
                  <p className="page-placeholder">{t('changelog.loading')}</p>
                )}

                {online && !error && releases && releases.length === 0 && (
                  <p className="page-placeholder">{t('changelog.empty')}</p>
                )}

                {!error &&
                  releases &&
                  releases.map((release, index) => {
                    const groupKey = groupLabelKey(index);
                    const showGroupLabel = index <= 2;

                    return (
                      <div className="changelog-group" key={release.tag_name}>
                        {showGroupLabel && (
                          <span className="changelog-group-label">
                            {t(groupKey)}
                          </span>
                        )}
                        <button
                          className="changelog-entry"
                          onClick={() => setSelected(release)}
                        >
                          <div className="changelog-entry-top">
                            <span className="changelog-entry-version">
                              {release.tag_name}
                            </span>
                            <span className="changelog-entry-date">
                              {formatDate(release.published_at)}
                            </span>
                          </div>
                          <h3 className="changelog-entry-title">
                            {extractTitle(release)}
                          </h3>
                          <p className="changelog-entry-excerpt">
                            {extractExcerpt(release.body)}
                          </p>
                        </button>
                      </div>
                    );
                  })}
              </div>
            ),
          }))}
        />
      </Panel>

      {selected && (
        <Modal
          variant="close"
          size="wide"
          open={selected as unknown as boolean}
          title={extractTitle(selected)}
          onClose={() => setSelected(null)}
        >
          <p className="changelog-modal-date">
            {formatDate(selected.published_at)}
          </p>
          <div className="changelog-entry-body">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>
              {selected.body}
            </ReactMarkdown>
          </div>
        </Modal>
      )}
    </div>
  );
}

export default ChangelogPage;
