/**
 * Clean Modern Light Forum - Interactive Logic (Vanilla ES6 JS)
 * Target: tmp/demos/clean-light-forum/app.js
 */

document.addEventListener('DOMContentLoaded', () => {
  initVoteButtons();
  initBookmarkButtons();
  initModalManager();
  initCommentSystem();
  initCategoryPills();
});

/**
 * 1. Toast Notification Helper
 */
function showToast(message, type = 'success') {
  const existingToast = document.querySelector('.toast-notification');
  if (existingToast) existingToast.remove();

  const toast = document.createElement('div');
  toast.className = `toast-notification toast-${type}`;
  toast.style.cssText = `
    position: fixed;
    bottom: 24px;
    right: 24px;
    z-index: 300;
    padding: 12px 20px;
    background: #0f172a;
    color: #ffffff;
    border-radius: 8px;
    font-size: 0.875rem;
    font-weight: 500;
    box-shadow: 0 10px 15px -3px rgba(0,0,0,0.1);
    display: flex;
    align-items: center;
    gap: 8px;
    transition: transform 0.2s ease, opacity 0.2s ease;
    transform: translateY(10px);
    opacity: 0;
  `;
  toast.innerHTML = `<span>${message}</span>`;
  document.body.appendChild(toast);

  requestAnimationFrame(() => {
    toast.style.transform = 'translateY(0)';
    toast.style.opacity = '1';
  });

  setTimeout(() => {
    toast.style.transform = 'translateY(10px)';
    toast.style.opacity = '0';
    setTimeout(() => toast.remove(), 200);
  }, 3000);
}

/**
 * 2. Vote Buttons (Upvote / Downvote)
 */
function initVoteButtons() {
  document.body.addEventListener('click', (e) => {
    const voteBtn = e.target.closest('.btn-vote');
    if (!voteBtn) return;

    const countSpan = voteBtn.querySelector('.vote-count');
    let count = parseInt(countSpan?.textContent || '0', 10);
    const isActive = voteBtn.classList.contains('active');

    if (isActive) {
      voteBtn.classList.remove('active');
      if (countSpan) countSpan.textContent = count - 1;
      showToast('已取消点赞');
    } else {
      voteBtn.classList.add('active');
      if (countSpan) countSpan.textContent = count + 1;
      showToast('点赞成功 +1');
    }
  });
}

/**
 * 3. Bookmark Toggle
 */
function initBookmarkButtons() {
  document.body.addEventListener('click', (e) => {
    const bookmarkBtn = e.target.closest('.btn-bookmark');
    if (!bookmarkBtn) return;

    const isActive = bookmarkBtn.classList.contains('active');
    if (isActive) {
      bookmarkBtn.classList.remove('active');
      bookmarkBtn.style.color = 'var(--text-muted)';
      showToast('已取消收藏');
    } else {
      bookmarkBtn.classList.add('active');
      bookmarkBtn.style.color = 'var(--accent)';
      showToast('已加入我的收藏');
    }
  });
}

/**
 * 4. Modal Manager (New Post Dialog)
 */
function initModalManager() {
  const modal = document.getElementById('newPostModal');
  if (!modal) return;

  const openBtns = document.querySelectorAll('.btn-open-modal');
  const closeBtns = modal.querySelectorAll('.btn-close-modal');
  const form = document.getElementById('newPostForm');

  const openModal = () => modal.classList.add('active');
  const closeModal = () => modal.classList.remove('active');

  openBtns.forEach(btn => btn.addEventListener('click', openModal));
  closeBtns.forEach(btn => btn.addEventListener('click', closeModal));

  modal.addEventListener('click', (e) => {
    if (e.target === modal) closeModal();
  });

  if (form) {
    form.addEventListener('submit', (e) => {
      e.preventDefault();
      const title = form.querySelector('[name="title"]')?.value;
      const category = form.querySelector('[name="category"]')?.value;

      if (!title) {
        showToast('请输入帖子标题', 'warning');
        return;
      }

      closeModal();
      form.reset();
      showToast(`帖子 "${title}" 发布成功！`, 'success');

      // Append new post item into feed if feed container exists
      const feedContainer = document.querySelector('.feed-container');
      if (feedContainer) {
        const newCard = createPostCardElement(title, category);
        feedContainer.insertAdjacentElement('afterbegin', newCard);
      }
    });
  }
}

/**
 * Helper to generate a new feed post card element
 */
function createPostCardElement(title, category) {
  const card = document.createElement('div');
  card.className = 'card post-card';
  card.innerHTML = `
    <div class="post-meta">
      <div class="author-info">
        <img class="avatar" src="https://api.dicebear.com/7.x/avataaars/svg?seed=CurrentWorker" alt="Avatar">
        <div>
          <span class="author-name">当前用户</span>
          <span class="author-badge">作者</span>
        </div>
      </div>
      <span class="post-time">刚刚</span>
    </div>
    <h3 class="post-title"><a href="post.html">${escapeHTML(title)}</a></h3>
    <p class="post-excerpt">这是您刚刚发布的新帖子，包含了讨论的核心摘要与标签...</p>
    <div class="tag-list">
      <span class="tag">${escapeHTML(category || 'General')}</span>
      <span class="tag">#新发布</span>
    </div>
    <div class="post-footer">
      <div class="action-group">
        <button class="btn-vote">
          <svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M12 19V5M5 12l7-7 7 7"/></svg>
          <span class="vote-count">1</span>
        </button>
        <span class="action-item">
          <svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
          0 评论
        </span>
      </div>
      <span>浏览 1</span>
    </div>
  `;
  return card;
}

/**
 * 5. Multi-Level Comment System
 */
function initCommentSystem() {
  const primaryForm = document.getElementById('primaryCommentForm');
  const commentList = document.getElementById('commentList');

  // Submit Top-Level Comment
  if (primaryForm && commentList) {
    primaryForm.addEventListener('submit', (e) => {
      e.preventDefault();
      const textarea = primaryForm.querySelector('textarea');
      const content = textarea.value.trim();

      if (!content) {
        showToast('请输入评论内容', 'warning');
        return;
      }

      const commentNode = createCommentElement({
        author: '极客开发者',
        badge: 'LV.3',
        time: '刚刚',
        content: content,
        isNested: false
      });

      commentList.insertAdjacentElement('afterbegin', commentNode);
      textarea.value = '';
      showToast('评论发布成功！');

      // Update count header
      const countEl = document.querySelector('.comments-count');
      if (countEl) {
        const currentCount = parseInt(countEl.textContent.match(/\d+/) || '0', 10);
        countEl.textContent = `全部评论 (${currentCount + 1})`;
      }
    });
  }

  // Handle Reply Button Triggers
  document.body.addEventListener('click', (e) => {
    const replyBtn = e.target.closest('.btn-reply');
    if (!replyBtn) return;

    const commentCard = replyBtn.closest('.comment-card');
    if (!commentCard) return;

    let existingReplyBox = commentCard.querySelector('.inline-reply-box');
    if (existingReplyBox) {
      existingReplyBox.remove();
      return;
    }

    const authorName = commentCard.querySelector('.author-name')?.textContent || '用户';
    const inlineBox = document.createElement('div');
    inlineBox.className = 'inline-reply-box';
    inlineBox.innerHTML = `
      <input type="text" class="inline-reply-input" placeholder="回复 @${escapeHTML(authorName)}...">
      <div style="display:flex; justify-content:flex-end; gap:8px;">
        <button class="btn btn-secondary btn-sm btn-cancel-reply">取消</button>
        <button class="btn btn-primary btn-sm btn-submit-reply">发表回复</button>
      </div>
    `;

    commentCard.appendChild(inlineBox);
    const input = inlineBox.querySelector('.inline-reply-input');
    input.focus();

    // Cancel inline reply
    inlineBox.querySelector('.btn-cancel-reply').addEventListener('click', () => {
      inlineBox.remove();
    });

    // Submit inline reply
    inlineBox.querySelector('.btn-submit-reply').addEventListener('click', () => {
      const replyContent = input.value.trim();
      if (!replyContent) {
        showToast('请输入回复内容', 'warning');
        return;
      }

      // Ensure nested tree container exists
      let nestedTree = commentCard.querySelector('.comment-nested-tree');
      if (!nestedTree) {
        nestedTree = document.createElement('div');
        nestedTree.className = 'comment-nested-tree';
        commentCard.appendChild(nestedTree);
      }

      const nestedCommentNode = createCommentElement({
        author: '我 (访客)',
        badge: '互动中',
        time: '刚刚',
        content: replyContent,
        replyTo: authorName,
        isNested: true
      });

      nestedTree.appendChild(nestedCommentNode);
      inlineBox.remove();
      showToast('回复发表成功！');
    });
  });
}

/**
 * Helper to generate Comment Element
 */
function createCommentElement({ author, badge, time, content, replyTo, isNested }) {
  const card = document.createElement('div');
  card.className = `comment-card ${isNested ? 'nested' : ''}`;
  card.innerHTML = `
    <div class="comment-header">
      <div class="author-info">
        <img class="avatar" src="https://api.dicebear.com/7.x/avataaars/svg?seed=${encodeURIComponent(author)}" alt="Avatar">
        <div>
          <span class="author-name">${escapeHTML(author)}</span>
          <span class="author-badge">${escapeHTML(badge)}</span>
        </div>
      </div>
      <span class="post-time">${escapeHTML(time)}</span>
    </div>
    <div class="comment-content">
      ${replyTo ? `<span class="reply-to-tag">@${escapeHTML(replyTo)}</span> ` : ''}
      ${escapeHTML(content)}
    </div>
    <div class="comment-footer">
      <button class="btn-vote">
        <svg width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M12 19V5M5 12l7-7 7 7"/></svg>
        <span class="vote-count">0</span>
      </button>
      <button class="btn-reply">
        <svg width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
        回复
      </button>
    </div>
  `;
  return card;
}

/**
 * 6. Category Pill Filter Active Switch
 */
function initCategoryPills() {
  const pills = document.querySelectorAll('.pill');
  pills.forEach(pill => {
    pill.addEventListener('click', () => {
      pills.forEach(p => p.classList.remove('active'));
      pill.classList.add('active');
      showToast(`已筛选: ${pill.textContent.trim()}`);
    });
  });
}

function escapeHTML(str) {
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}
