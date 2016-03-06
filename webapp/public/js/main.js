'use strict';
$(function () {
  $('#isu-post-more-btn').on('click', function () {
    $('#isu-post-more').addClass('loading');
    $.ajax({
      type: 'GET',
      url: '/posts',
      data: {
        max_created_at: $('.isu-post:last').attr('data-created-at')
      },
      dataType: 'html'
    }).done(function (html) {
      $(html).find('.isu-post').each(function() {
        var id = $(this).attr('id');
        if ($('#' + id).length === 0) {
          $('.isu-posts').append($(this).clone());
        }
      });
      $('time.timeago').timeago();
      $('#isu-post-more').removeClass('loading');
    });
  });

  $('time.timeago').timeago();
});
