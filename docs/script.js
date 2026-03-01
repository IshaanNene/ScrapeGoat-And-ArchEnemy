/* ═══════════════════════════════════════════
   ScrapeGoat Docs — Interactivity
   ═══════════════════════════════════════════ */

document.addEventListener('DOMContentLoaded', () => {
  initThemeToggle();
  initMobileNav();
  initCopyButtons();
  initFAQ();
  initScrollAnimations();
  initActiveNavigation();
  initBackToTop();
  initCommandTabs();
});

/* ── Theme Toggle ── */
function initThemeToggle() {
  const btn = document.getElementById('theme-toggle');
  const saved = localStorage.getItem('scrapegoat-theme');
  if (saved === 'dark') {
    document.documentElement.setAttribute('data-theme', 'dark');
    btn.textContent = '☀️';
  }

  btn.addEventListener('click', () => {
    const isDark = document.documentElement.getAttribute('data-theme') === 'dark';
    if (isDark) {
      document.documentElement.removeAttribute('data-theme');
      btn.textContent = '🌙';
      localStorage.setItem('scrapegoat-theme', 'light');
    } else {
      document.documentElement.setAttribute('data-theme', 'dark');
      btn.textContent = '☀️';
      localStorage.setItem('scrapegoat-theme', 'dark');
    }
  });
}

/* ── Mobile Navigation ── */
function initMobileNav() {
  const hamburger = document.getElementById('hamburger');
  const mobileNav = document.getElementById('mobile-nav');

  hamburger.addEventListener('click', () => {
    mobileNav.classList.toggle('open');
    const spans = hamburger.querySelectorAll('span');
    if (mobileNav.classList.contains('open')) {
      spans[0].style.transform = 'rotate(45deg) translate(5px, 5px)';
      spans[1].style.opacity = '0';
      spans[2].style.transform = 'rotate(-45deg) translate(6px, -6px)';
    } else {
      spans[0].style.transform = '';
      spans[1].style.opacity = '1';
      spans[2].style.transform = '';
    }
  });

  // Close on link click
  mobileNav.querySelectorAll('a').forEach(link => {
    link.addEventListener('click', () => {
      mobileNav.classList.remove('open');
      const spans = hamburger.querySelectorAll('span');
      spans[0].style.transform = '';
      spans[1].style.opacity = '1';
      spans[2].style.transform = '';
    });
  });
}

/* ── Copy to Clipboard ── */
function initCopyButtons() {
  document.querySelectorAll('.code-block__copy').forEach(btn => {
    btn.addEventListener('click', () => {
      const codeBlock = btn.closest('.code-block');
      const code = codeBlock.querySelector('code').textContent;

      navigator.clipboard.writeText(code).then(() => {
        btn.textContent = '✓ Copied!';
        btn.classList.add('code-block__copy--copied');
        setTimeout(() => {
          btn.textContent = 'Copy';
          btn.classList.remove('code-block__copy--copied');
        }, 2000);
      }).catch(() => {
        // Fallback
        const textarea = document.createElement('textarea');
        textarea.value = code;
        document.body.appendChild(textarea);
        textarea.select();
        document.execCommand('copy');
        document.body.removeChild(textarea);
        btn.textContent = '✓ Copied!';
        btn.classList.add('code-block__copy--copied');
        setTimeout(() => {
          btn.textContent = 'Copy';
          btn.classList.remove('code-block__copy--copied');
        }, 2000);
      });
    });
  });
}

/* ── FAQ Accordion ── */
function initFAQ() {
  document.querySelectorAll('.faq-question').forEach(btn => {
    btn.addEventListener('click', () => {
      const item = btn.closest('.faq-item');
      const answer = item.querySelector('.faq-answer');
      const isOpen = item.classList.contains('open');

      // Close all others
      document.querySelectorAll('.faq-item.open').forEach(openItem => {
        if (openItem !== item) {
          openItem.classList.remove('open');
          openItem.querySelector('.faq-answer').style.maxHeight = '0';
        }
      });

      if (isOpen) {
        item.classList.remove('open');
        answer.style.maxHeight = '0';
      } else {
        item.classList.add('open');
        answer.style.maxHeight = answer.scrollHeight + 'px';
      }
    });
  });
}

/* ── Scroll Animations ── */
function initScrollAnimations() {
  const observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
      if (entry.isIntersecting) {
        entry.target.classList.add('visible');
      }
    });
  }, { threshold: 0.1, rootMargin: '0px 0px -40px 0px' });

  document.querySelectorAll('.fade-in').forEach(el => observer.observe(el));
}

/* ── Active Navigation Highlighting ── */
function initActiveNavigation() {
  const sections = document.querySelectorAll('section[id]');
  const navLinks = document.querySelectorAll('.navbar__links a');

  const observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
      if (entry.isIntersecting) {
        const id = entry.target.id;
        navLinks.forEach(link => {
          link.classList.toggle('active', link.getAttribute('href') === `#${id}`);
        });
      }
    });
  }, { rootMargin: '-30% 0px -60% 0px' });

  sections.forEach(section => observer.observe(section));
}

/* ── Back to Top ── */
function initBackToTop() {
  const btn = document.getElementById('back-to-top');

  window.addEventListener('scroll', () => {
    btn.classList.toggle('visible', window.scrollY > 500);
  });

  btn.addEventListener('click', () => {
    window.scrollTo({ top: 0, behavior: 'smooth' });
  });
}

/* ── Command Tabs ── */
function initCommandTabs() {
  const tabs = document.querySelectorAll('.commands-tab');
  const panels = document.querySelectorAll('.command-panel');

  tabs.forEach(tab => {
    tab.addEventListener('click', () => {
      const target = tab.dataset.tab;

      tabs.forEach(t => t.classList.remove('active'));
      panels.forEach(p => p.classList.remove('active'));

      tab.classList.add('active');
      document.getElementById(`panel-${target}`).classList.add('active');
    });
  });
}
