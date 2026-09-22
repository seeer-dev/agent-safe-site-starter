(() => {
  const form = document.querySelector('[data-contact-form]');
  if (!form) return;

  const apiBase = document.documentElement.dataset.apiBase || '';
  const status = form.querySelector('[data-form-status]');
  // When a Turnstile site key is configured the managed widget mints a
  // single-use token into the hidden cf-turnstile-response field; the API
  // requires it as turnstile_token. With no key (local development) the
  // form submits without one.
  const turnstileEnabled = Boolean(form.dataset.turnstileSitekey);

  form.addEventListener('submit', async (event) => {
    event.preventDefault();
    status.textContent = '送出中…';
    const data = Object.fromEntries(new FormData(form).entries());
    const token = data['cf-turnstile-response'] || '';
    delete data['cf-turnstile-response'];
    if (turnstileEnabled && !token) {
      status.textContent = '請先完成驗證。';
      return;
    }
    if (token) data.turnstile_token = token;

    try {
      const response = await fetch(`${apiBase}/api/contact`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });
      const body = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(body.error || `HTTP ${response.status}`);
      form.reset();
      if (turnstileEnabled && window.turnstile) window.turnstile.reset();
      status.textContent = '已送出。';
    } catch (error) {
      if (turnstileEnabled && window.turnstile) window.turnstile.reset();
      status.textContent = `送出失敗：${error.message}`;
    }
  });
})();
