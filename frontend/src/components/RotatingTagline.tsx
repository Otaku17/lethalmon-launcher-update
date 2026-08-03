import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import './RotatingTagline.css';

const INTERVAL_MS = 5000;

function RotatingTagline() {
  const { t } = useTranslation();
  const taglines = t('home.taglines', { returnObjects: true }) as string[];
  const [index, setIndex] = useState(0);

  useEffect(() => {
    const interval = setInterval(() => {
      setIndex((i) => (i + 1) % taglines.length);
    }, INTERVAL_MS);
    return () => clearInterval(interval);
  }, [taglines.length]);

  return (
    <p key={index} className="rotating-tagline">
      {taglines[index]}
    </p>
  );
}

export default RotatingTagline;
