'use strict';

// cf: https://github.com/hustcc/timeago.js
timeago.register('ja', (number, index) => {
  return [
    ['すこし前', 'すぐに'],
    ['%s秒前', '%s秒以内'],
    ['1分前', '1分以内'],
    ['%s分前', '%s分以内'],
    ['1時間前', '1時間以内'],
    ['%s時間前', '%s時間以内'],
    ['1日前', '1日以内'],
    ['%s日前', '%s日以内'],
    ['1週間前', '1週間以内'],
    ['%s週間前', '%s週間以内'],
    ['1ヶ月前', '1ヶ月以内'],
    ['%sヶ月前', '%sヶ月以内'],
    ['1年前', '1年以内'],
    ['%s年前', '%s年以内'],
  ][index];
})

document.addEventListener('DOMContentLoaded', () => {
  timeago.render(document.querySelectorAll('time.timeago'), 'ja');

  const btn = document.getElementById('isu-post-more-btn');
  const postMore = document.getElementById('isu-post-more');

  if (!btn) {
    return;
  }

  btn.addEventListener('click', () => {
    postMore.classList.add('loading');
    const posts = document.querySelectorAll('.isu-post');
    const lastEl = posts[posts.length-1];
    const maxCreatedAt = lastEl.dataset.createdAt;
    fetch(`/posts?max_created_at=${encodeURIComponent(maxCreatedAt)}`, {
      method: 'GET',
    }).then(response => {
      if (!response.ok) {
        throw new Error('Network response was not ok');
      }
      return response.text();
    }).then(text => {
      const parser = new DOMParser();
      const doc = parser.parseFromString(text, "text/html");
      doc.querySelectorAll('.isu-post').forEach((el) => {
        const id = el.getAttribute('id');
        if (!document.getElementById(id)) {
          lastEl.parentElement.append(el);
        }
      });
      timeago.render(document.querySelectorAll('time.timeago'), 'ja');
      postMore.classList.remove('loading');
    });
  });
});
