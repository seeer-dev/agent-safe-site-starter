(() => {
  const form = document.querySelector('[data-contact-form]');
  if (!form) return;

  const apiBase = document.documentElement.dataset.apiBase || '';
  const status = form.querySelector('[data-form-status]');

  form.addEventListener('submit', async (event) => {
    event.preventDefault();
    status.textContent = '送出中…';
    const data = Object.fromEntries(new FormData(form).entries());

    try {
      const response = await fetch(`${apiBase}/api/contact`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });
      const body = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(body.error || `HTTP ${response.status}`);
      form.reset();
      status.textContent = '已送出。';
    } catch (error) {
      status.textContent = `送出失敗：${error.message}`;
    }
  });
})();
