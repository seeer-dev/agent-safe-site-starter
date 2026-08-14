<script setup lang="ts">
import Button from '@/components/ui/Button.vue'

const states = [
  {
    title: '載入中',
    type: 'loading',
    icon: '',
  },
  {
    title: '空清單',
    type: 'empty',
    icon: 'M4 7h16v13H4z;M4 7l4-4h8l4 4',
    body: '還沒有優惠活動',
    desc: '建立第一個折扣碼，前台結帳才會出現優惠欄位。',
    action: '新增優惠',
  },
  {
    title: '載入失敗',
    type: 'error',
    icon: 'M12 9v4;M12 17h.01;circle 12 12 9',
    body: '讀取清單失敗',
    desc: '連線逾時，通常是暫時的。',
    action: '重試',
    danger: true,
  },
  {
    title: '權限不足',
    type: 'forbidden',
    icon: 'M4 10 H20 V20 H4 Z;M8 10 V7 a4 4 0 0 1 8 0 v3',
    body: '這頁在你的角色下不會出現',
    desc: '本站的 deniedMode 是 hide：沒有權限的入口直接不顯示在側欄，不是停用。深連結才會看到這個畫面。',
    warn: true,
  },
]
</script>

<template>
  <div class="pagehd">
    <div>
      <h1>五狀態參考</h1>
      <div class="sub">通用清單殼在非理想情況下的樣子</div>
    </div>
  </div>

  <div class="cols">
    <!-- Loading -->
    <section class="panel">
      <div class="phd"><h3>載入中</h3></div>
      <div style="padding:16px">
        <div style="margin:8px 0;height:11px;width:70%;border-radius:4px;background:var(--surface-3)" />
        <div style="margin:8px 0;height:11px;border-radius:4px;background:var(--surface-3)" />
        <div style="margin:8px 0;height:11px;width:85%;border-radius:4px;background:var(--surface-3)" />
      </div>
    </section>

    <!-- Empty -->
    <section class="panel">
      <div class="phd"><h3>空清單</h3></div>
      <div class="emptybox">
        <div class="ic">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">
            <path d="M4 7h16v13H4z" /><path d="M4 7l4-4h8l4 4" />
          </svg>
        </div>
        <b>還沒有優惠活動</b>
        <p>建立第一個折扣碼，前台結帳才會出現優惠欄位。</p>
        <Button variant="pri">新增優惠</Button>
      </div>
    </section>

    <!-- Error -->
    <section class="panel">
      <div class="phd"><h3>載入失敗</h3></div>
      <div class="emptybox">
        <div class="ic" style="background:var(--red-bg);color:var(--red)">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 9v4M12 17h.01" /><circle cx="12" cy="12" r="9" />
          </svg>
        </div>
        <b>讀取清單失敗</b>
        <p>連線逾時，通常是暫時的。</p>
        <Button>重試</Button>
      </div>
    </section>

    <!-- Forbidden -->
    <section class="panel">
      <div class="phd"><h3>權限不足</h3></div>
      <div class="emptybox">
        <div class="ic" style="background:var(--amber-bg);color:var(--amber)">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">
            <rect x="4" y="10" width="16" height="10" rx="2" /><path d="M8 10V7a4 4 0 0 1 8 0v3" />
          </svg>
        </div>
        <b>這頁在你的角色下不會出現</b>
        <p>本站的 deniedMode 是 hide：沒有權限的入口直接不顯示在側欄，不是停用。深連結才會看到這個畫面。</p>
      </div>
    </section>
  </div>
</template>
