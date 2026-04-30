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
  let subjectsSignature = '';
  let entriesSignature = '';
  let loading = true;
  let saving = false;
  let error = '';
  let midnightTimer;
  let refreshTimer;
  let today = startOfToday();

  let isMobile = window.matchMedia('(max-width: 720px)').matches;

  function handleResize() {
    isMobile = window.matchMedia('(max-width: 720px)').matches;
  }

  let colorInput;
  let colorTarget = '';
  let selectedSubject = '';
  let editOpen = false;
  let editDate = null;
  let editSubject = '';
  let editMinutes = '';

  function openColorPicker(subjectName) {
    colorTarget = subjectName;
    const subject = subjects.find((s) => s.name === subjectName);
    if (!subject) return;
    colorInput.value = subject.color;
    colorInput.click();
  }

  async function onColorChange(e) {
    const newColor = e.target.value;
    if (!colorTarget) {
      return;
    }
    saving = true;
    error = '';
    try {
      const res = await fetch(`/api/subjects/${encodeURIComponent(colorTarget)}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ color: newColor }),
      });
      if (!res.ok) {
        throw new Error(await res.text() || 'Failed to save color');
      }
      const updated = await res.json();
      subjects = subjects.map((subject) => subject.name === updated.name ? updated : subject);
    } catch (err) {
      error = err.message || 'Failed to save color';
    } finally {
      saving = false;
    }
  }

  async function addSubject() {
    const name = window.prompt('New topic name');
    if (name === null) {
      return;
    }
    saving = true;
    error = '';
    try {
      const res = await fetch('/api/subjects', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name }),
      });
      if (!res.ok) {
        throw new Error(await res.text() || 'Failed to add topic');
      }
      await load();
    } catch (err) {
      error = err.message || 'Failed to add topic';
    } finally {
      saving = false;
    }
  }

  function toggleSubjectFilter(subjectName) {
    selectedSubject = selectedSubject === subjectName ? '' : subjectName;
  }

  function clearSubjectFilter() {
    selectedSubject = '';
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
    return dates[item.dataIndex].toLocaleDateString(undefined, {
      weekday: 'long',
      day: 'numeric',
      month: 'long',
    });
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

  function entriesForSubjectAndDate(subjectName, date) {
    const dateStr = formatDate(date);
    return entries.filter((entry) => entry.subject === subjectName && entry.date === dateStr);
  }

  function totalMinutesForSubjectAndDate(subjectName, date) {
    return entriesForSubjectAndDate(subjectName, date).reduce((sum, entry) => sum + entry.minutes, 0);
  }

  function syncEditMinutes() {
    if (!editDate || !editSubject) {
      editMinutes = '';
      return;
    }
    editMinutes = `${totalMinutesForSubjectAndDate(editSubject, editDate)}`;
  }

  function openEditModal(date, preferredSubject = '') {
    if (!date) {
      return;
    }
    editDate = date;
    editSubject = preferredSubject || selectedSubject || subjects[0]?.name || '';
    syncEditMinutes();
    editOpen = true;
  }

  function closeEditModal() {
    editOpen = false;
  }

  async function saveStudyMinutes() {
    const nextMinutes = Number.parseInt(String(editMinutes ?? '').trim(), 10);
    if (!Number.isInteger(nextMinutes) || nextMinutes < 0) {
      error = 'Minutes must be a whole number (0 or more).';
      return;
    }
    if (!editDate || !editSubject) {
      error = 'Pick a date and topic.';
      return;
    }

    saving = true;
    error = '';
    try {
      const existing = entriesForSubjectAndDate(editSubject, editDate);
      await Promise.all(existing.map(async (entry) => {
        const res = await fetch(`/api/entries/${entry.id}`, { method: 'DELETE' });
        if (!res.ok) {
          throw new Error(await res.text() || 'Failed to update entry');
        }
      }));

      if (nextMinutes > 0) {
        const res = await fetch('/api/entries', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            date: formatDate(editDate),
            subject: editSubject,
            minutes: nextMinutes,
          }),
        });
        if (!res.ok) {
          throw new Error(await res.text() || 'Failed to update entry');
        }
      }

      await load();
      closeEditModal();
    } catch (err) {
      error = err.message || 'Failed to update entry';
    } finally {
      saving = false;
    }
  }

  function onChartClick(event, _elements, chart) {
    const nearest = chart.getElementsAtEventForMode(event, 'index', { intersect: false }, false);
    const item = nearest?.[0];

    let index = item?.index;
    if (index === undefined) {
      const xScale = chart?.scales?.x;
      const x = event?.x;
      if (xScale && Number.isFinite(x)) {
        index = Math.round(xScale.getValueForPixel(x));
      }
    }
    if (!Number.isInteger(index) || index < 0 || index >= dates.length) {
      return;
    }

    const date = dates[index];
    const preferredSubject = item ? (filteredDatasets[item.datasetIndex]?.label || '') : '';
    openEditModal(date, preferredSubject);
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

      const nextToday = startOfToday();
      if (formatDate(nextToday) !== formatDate(today)) {
        today = nextToday;
      }

      const nextSubjects = await subjectsRes.json();
      const nextEntries = await entriesRes.json();
      const nextSubjectsSignature = JSON.stringify(nextSubjects);
      const nextEntriesSignature = JSON.stringify(nextEntries);

      if (nextSubjectsSignature !== subjectsSignature) {
        subjects = nextSubjects;
        subjectsSignature = nextSubjectsSignature;
      }
      if (nextEntriesSignature !== entriesSignature) {
        entries = nextEntries;
        entriesSignature = nextEntriesSignature;
      }

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
    refreshTimer = setInterval(load, 20 * 60 * 1000);
    window.addEventListener('resize', handleResize);
    return () => {
      clearTimeout(midnightTimer);
      clearInterval(refreshTimer);
      window.removeEventListener('resize', handleResize);
    };
  });

  $: if (selectedSubject && !subjects.some((subject) => subject.name === selectedSubject)) {
    selectedSubject = '';
  }
  $: allDates = rollingDates();
  $: dates = isMobile ? allDates.slice(-10) : allDates;
  $: labels = dates.map(shortLabel);
  $: totals = entries.reduce((acc, entry) => {
    const key = `${entry.date}:${entry.subject}`;
    acc.set(key, (acc.get(key) || 0) + entry.minutes);
    return acc;
  }, new Map());
  $: datasets = subjects.map((subject) => ({
    label: subject.name,
    data: dates.map((date) => {
      const value = totals.get(`${formatDate(date)}:${subject.name}`);
      return value === undefined ? null : value;
    }),
    backgroundColor: subject.color,
    borderColor: dates.map((date) => formatDate(date) === formatDate(today) ? 'rgba(248,250,252,0.9)' : 'rgba(0,0,0,0)'),
    borderWidth: dates.map((date) => formatDate(date) === formatDate(today) ? 2 : 0),
    borderSkipped: false,
    borderRadius: 4,
    categoryPercentage: 0.9,
    barPercentage: 1,
    barThickness: isMobile ? 8 : 12,
    maxBarThickness: isMobile ? 8 : 12,
    skipNull: true,
  }));
  $: filteredDatasets = selectedSubject
    ? datasets.filter((dataset) => dataset.label === selectedSubject)
    : datasets;
  $: data = { labels, datasets: filteredDatasets };
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
    onClick: onChartClick,
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
  <div class="card">
    <div class="header">
      <h2>
        <button class="title-link" on:click={clearSubjectFilter}>Study Blocks</button>
      </h2>
      <p>Estimated study minutes by subject (click a day to edit)</p>
    </div>

    {#if error}
      <div class="state error">{error}</div>
    {:else if loading}
      <div class="state">Loading…</div>
    {:else}
      {#if saving}
        <div class="state">Saving…</div>
      {/if}
      <div class="chart-wrap">
        <Bar {data} {options} />
      </div>
      <div class="legend">
        {#each subjects as subject}
          <div class="legend-item" class:active={selectedSubject === subject.name}>
            <span
              class="swatch"
              role="button"
              tabindex="0"
              style={`background: ${subject.color}`}
              on:click={() => openColorPicker(subject.name)}
              on:keydown={(e) => { if (e.key === 'Enter' || e.key === ' ') openColorPicker(subject.name); }}
            ></span>
            <span
              class="subject-name"
              role="button"
              tabindex="0"
              on:click={() => toggleSubjectFilter(subject.name)}
              on:keydown={(e) => { if (e.key === 'Enter' || e.key === ' ') toggleSubjectFilter(subject.name); }}
            >{subject.name}</span>
          </div>
        {/each}
        <button class="add-topic" on:click={addSubject}>add</button>
      </div>
    {/if}

    {#if editOpen}
      <div
        class="modal-backdrop"
        role="button"
        tabindex="0"
        on:click|self={closeEditModal}
        on:keydown={(e) => { if (e.key === 'Escape') closeEditModal(); }}
      >
        <div class="modal" role="dialog" aria-modal="true" tabindex="-1">
          <h3>
            {#if editDate}
              {editDate.toLocaleDateString(undefined, { weekday: 'long', day: 'numeric', month: 'long' })}
            {/if}
          </h3>
          <label>
            Topic
            <select bind:value={editSubject} on:change={syncEditMinutes}>
              {#each subjects as subject}
                <option value={subject.name}>{subject.name}</option>
              {/each}
            </select>
          </label>
          <label>
            Minutes
            <input type="number" min="0" step="1" bind:value={editMinutes} />
          </label>
          <div class="modal-actions">
            <button type="button" on:click={closeEditModal}>Cancel</button>
            <button type="button" on:click={saveStudyMinutes}>Save</button>
          </div>
        </div>
      </div>
    {/if}
  </div>
</div>

<style>
  :global(html),
  :global(body) {
    margin: 0;
    font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    background: #020617;
    color: #f8fafc;
    min-height: 100vh;
  }

  :global(body) {
    background:
      radial-gradient(circle at top, rgba(71, 85, 105, 0.18), transparent 35%),
      linear-gradient(180deg, #020617, #0f172a);
  }

  :global(*) {
    box-sizing: border-box;
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

  .title-link {
    all: unset;
    cursor: pointer;
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
    align-items: center;
    gap: 12px 16px;
  }

  .legend-item {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    color: #cbd5e1;
  }

  .legend-item.active .subject-name {
    color: #f8fafc;
    text-decoration: underline;
    text-underline-offset: 3px;
  }

  .subject-name {
    cursor: pointer;
  }

  .add-topic {
    margin-left: auto;
    border: none;
    background: transparent;
    color: #94a3b8;
    cursor: pointer;
    text-transform: lowercase;
    font-size: 0.9rem;
  }

  .add-topic:hover {
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

  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(2, 6, 23, 0.55);
    display: grid;
    place-items: center;
    z-index: 20;
  }

  .modal {
    width: min(360px, calc(100vw - 2rem));
    background: rgba(15, 23, 42, 0.95);
    border: 1px solid rgba(148, 163, 184, 0.2);
    border-radius: 14px;
    padding: 14px;
    display: grid;
    gap: 10px;
  }

  .modal h3 {
    margin: 0;
    font-size: 1rem;
  }

  .modal label {
    display: grid;
    gap: 6px;
    color: #cbd5e1;
    font-size: 0.9rem;
  }

  .modal select,
  .modal input {
    border: 1px solid rgba(148, 163, 184, 0.25);
    border-radius: 8px;
    padding: 8px;
    background: rgba(2, 6, 23, 0.7);
    color: #f8fafc;
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }

  .modal-actions button {
    border: 1px solid rgba(148, 163, 184, 0.25);
    border-radius: 8px;
    padding: 6px 10px;
    background: rgba(15, 23, 42, 0.7);
    color: #f8fafc;
    cursor: pointer;
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
