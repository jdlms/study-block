<script>
  import {
    Chart as ChartJS,
    BarElement,
    CategoryScale,
    LinearScale,
    Tooltip,
    Legend,
  } from 'chart.js';
  import { Bar } from 'svelte-chartjs';
  import { onMount } from 'svelte';

  ChartJS.register(BarElement, CategoryScale, LinearScale, Tooltip, Legend);

  let subjects = [];
  let entries = [];
  let loading = true;
  let error = '';
  let midnightTimer;
  let today = startOfToday();

  let colorOverrides = JSON.parse(localStorage.getItem('study-blocks-colors') || '{}');
  let colorInput;
  let colorTarget = '';

  function openColorPicker(subjectName) {
    colorTarget = subjectName;
    const subject = subjects.find((s) => s.name === subjectName);
    if (!subject) return;
    colorInput.value = colorOverrides[subjectName] || subject.color;
    colorInput.click();
  }

  function onColorChange(e) {
    const newColor = e.target.value;
    colorOverrides = { ...colorOverrides, [colorTarget]: newColor };
    localStorage.setItem('study-blocks-colors', JSON.stringify(colorOverrides));
  }

  function startOfToday() {
    const date = new Date();
    date.setHours(0, 0, 0, 0);
    return date;
  }

  function scheduleMidnightRefresh() {
    clearTimeout(midnightTimer);
    const nextMidnight = new Date(today);
    nextMidnight.setDate(nextMidnight.getDate() + 1);
    const delay = nextMidnight.getTime() - Date.now();
    midnightTimer = setTimeout(async () => {
      today = startOfToday();
      await load();
      scheduleMidnightRefresh();
    }, Math.max(delay, 1000));
  }

  function formatDate(date) {
    const year = date.getFullYear();
    const month = `${date.getMonth() + 1}`.padStart(2, '0');
    const day = `${date.getDate()}`.padStart(2, '0');
    return `${year}-${month}-${day}`;
  }

  function shortLabel(date) {
    return `${date.getDate()}`;
  }

  function tooltipTitle(items) {
    const item = items[0];
    if (!item) {
      return '';
    }
    return dates[item.dataIndex].toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
  }

  function rollingDates() {
    const dates = [];
    for (let i = 29; i >= 0; i -= 1) {
      const date = new Date(today);
      date.setDate(today.getDate() - i);
      dates.push(date);
    }
    return dates;
  }

  async function load() {
    const dates = rollingDates();
    const from = formatDate(dates[0]);
    const to = formatDate(dates[dates.length - 1]);

    try {
      const [subjectsRes, entriesRes] = await Promise.all([
        fetch('/api/subjects'),
        fetch(`/api/entries?from=${from}&to=${to}`),
      ]);

      if (!subjectsRes.ok || !entriesRes.ok) {
        throw new Error('Failed to load chart data');
      }

      today = startOfToday();
      subjects = await subjectsRes.json();
      entries = await entriesRes.json();
      error = '';
    } catch (err) {
      error = err.message || 'Failed to load chart data';
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    load();
    scheduleMidnightRefresh();
    return () => clearTimeout(midnightTimer);
  });

  $: dates = rollingDates();
  $: labels = dates.map(shortLabel);
  $: totals = entries.reduce((acc, entry) => {
    const key = `${entry.date}:${entry.subject}`;
    acc.set(key, (acc.get(key) || 0) + entry.minutes);
    return acc;
  }, new Map());
  $: datasets = subjects.map((subject) => ({
    label: subject.name,
    data: dates.map((date) => totals.get(`${formatDate(date)}:${subject.name}`) || 0),
    backgroundColor: colorOverrides[subject.name] || subject.color,
    borderColor: dates.map((date) => formatDate(date) === formatDate(today) ? 'rgba(248,250,252,0.9)' : 'rgba(0,0,0,0)'),
    borderWidth: dates.map((date) => formatDate(date) === formatDate(today) ? 2 : 0),
    borderSkipped: false,
    borderRadius: 4,
    categoryPercentage: 0.9,
    barPercentage: 1,
  }));
  $: data = { labels, datasets };
  $: options = {
    responsive: true,
    maintainAspectRatio: false,
    interaction: { mode: 'index', intersect: false },
    plugins: {
      legend: { display: false },
      tooltip: {
        callbacks: {
          title: tooltipTitle,
          label: (ctx) => `${ctx.dataset.label}: ${ctx.parsed.y} min`,
        },
      },
    },
    scales: {
      x: {
        stacked: false,
        offset: true,
        ticks: {
          color: '#94a3b8',
          maxRotation: 0,
          autoSkip: false,
          font: { size: 10 },
          callback: (value, index) => labels[index],
        },
        grid: { color: 'rgba(148,163,184,0.08)' },
      },
      y: {
        stacked: false,
        beginAtZero: true,
        max: 120,
        ticks: {
          color: '#94a3b8',
          stepSize: 20,
          callback: (value) => value === 0 ? '0' : `${value}`,
        },
        grid: { color: 'rgba(148,163,184,0.12)' },
      },
    },
  };
</script>

<svelte:head>
  <title>🧗🏻 Study Blocks</title>
</svelte:head>

<input
  type="color"
  bind:this={colorInput}
  on:input={onColorChange}
  style="position:absolute;width:0;height:0;opacity:0;pointer-events:none"
/>

<div class="shell">
  <section class="hero">
    <div class="logo" aria-hidden="true">🧗🏻</div>
  </section>
  <div class="card">
    <div class="header">
      <h2>Study Blocks</h2>
      <p>Estimated study minutes stacked by subject</p>
    </div>

    {#if error}
      <div class="state error">{error}</div>
    {:else if loading}
      <div class="state">Loading…</div>
    {:else}
      <div class="chart-wrap">
        <Bar {data} {options} />
      </div>
      <div class="legend">
        {#each subjects as subject}
          <div class="legend-item">
            <span
              class="swatch"
              role="button"
              tabindex="0"
              style={`background: ${colorOverrides[subject.name] || subject.color}`}
              on:click={() => openColorPicker(subject.name)}
              on:keydown={(e) => { if (e.key === 'Enter' || e.key === ' ') openColorPicker(subject.name); }}
            ></span>
            <span>{subject.name}</span>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

<style>
  :global(html),
  :global(body) {
    margin: 0;
    font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    background:
      radial-gradient(circle at top, rgba(71, 85, 105, 0.18), transparent 35%),
      linear-gradient(180deg, #020617, #0f172a);
    color: #f8fafc;
    min-height: 100vh;
  }

  :global(*) {
    box-sizing: border-box;
  }

  .hero {
    text-align: center;
    margin-bottom: 3rem;
  }

  .logo {
    font-size: 3rem;
    line-height: 1;
    margin-bottom: 0.75rem;
  }

  .shell {
    width: min(920px, calc(100vw - 2rem));
    margin: 0 auto;
    padding: 2rem 0 3rem;
  }

  .card {
    margin: 0 auto;
    padding: 1.25rem;
    border-radius: 24px;
    background: rgba(15, 23, 42, 0.82);
    border: 1px solid rgba(148, 163, 184, 0.14);
    backdrop-filter: blur(14px);
    box-shadow: 0 30px 80px rgba(2, 6, 23, 0.45);
  }

  .header h2 {
    margin: 0;
    font-size: 1.2rem;
  }

  .header p {
    margin: 0 0 1rem;
    color: #94a3b8;
  }

  .chart-wrap {
    height: min(65vh, 480px);
    min-height: 320px;
    margin-top: 20px;
  }

  .legend {
    margin-top: 20px;
    display: flex;
    flex-wrap: wrap;
    gap: 12px 16px;
  }

  .legend-item {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    color: #cbd5e1;
  }

  .swatch {
    width: 12px;
    height: 12px;
    border-radius: 999px;
    cursor: pointer;
    transition: transform 0.15s ease;
  }

  .swatch:hover {
    transform: scale(1.4);
  }

  .state {
    margin-top: 20px;
    color: #cbd5e1;
  }

  .error {
    color: #fca5a5;
  }

  @media (max-width: 720px) {
    .shell {
      width: calc(100vw - 1rem);
      padding: 1rem 0 2rem;
    }

    .card {
      padding: 16px;
      border-radius: 18px;
    }

    .chart-wrap {
      min-height: 280px;
    }
  }
</style>
