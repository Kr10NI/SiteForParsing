import { useState } from 'react';
import './App.css';

function App() {
  const [url, setUrl] = useState('');
  const [selectors, setSelectors] = useState<string[]>(['']);
  const [results, setResults] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSelectorChange = (idx: number, value: string) => {
    setSelectors((prev) => prev.map((s, i) => (i === idx ? value : s)));
  };

  const addSelector = () => setSelectors((prev) => [...prev, '']);
  const removeSelector = (idx: number) => setSelectors((prev) => prev.filter((_, i) => i !== idx));

  const handleParse = async () => {
    setLoading(true);
    setError('');
    setResults([]);
    try {
      const res = await fetch('/api/parse', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url, selectors: selectors.filter(Boolean) }),
      });
      if (!res.ok) throw new Error('Ошибка запроса');
      const data = await res.json();
      setResults(data.results || []);
    } catch (e: any) {
      setError(e.message || 'Ошибка');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="container">
      <h1>Парсер сайта</h1>
      <input
        type="text"
        placeholder="Введите URL сайта"
        value={url}
        onChange={e => setUrl(e.target.value)}
        style={{ width: '100%', marginBottom: 12 }}
      />
      <div>
        {selectors.map((selector, idx) => (
          <div key={idx} style={{ display: 'flex', marginBottom: 8 }}>
            <input
              type="text"
              placeholder="CSS селектор"
              value={selector}
              onChange={e => handleSelectorChange(idx, e.target.value)}
              style={{ flex: 1 }}
            />
            {selectors.length > 1 && (
              <button onClick={() => removeSelector(idx)} style={{ marginLeft: 8 }}>Удалить</button>
            )}
          </div>
        ))}
        <button onClick={addSelector} style={{ marginBottom: 16 }}>Добавить селектор</button>
      </div>
      <button onClick={handleParse} disabled={loading || !url || (selectors.filter(Boolean).length === 0 && !url.includes('sportmaster.ru'))}>
        {loading ? 'Парсинг...' : 'Парсить'}
      </button>
      {error && <div style={{ color: 'red', marginTop: 12 }}>{error}</div>}
      <div style={{ marginTop: 24 }}>
        {Array.isArray(results) && results.length > 0 && <h2>Результаты</h2>}
        {Array.isArray(results) && results.map((r, i) => (
          <div key={i} style={{ marginBottom: 16 }}>
            <strong>{r.selector}</strong>
            <ul>
              {Array.isArray(r.results) && r.results.map((text: string, j: number) => (
                <li key={j}>{text}</li>
              ))}
            </ul>
          </div>
        ))}
      </div>
    </div>
  );
}

export default App;
