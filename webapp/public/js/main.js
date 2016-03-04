$(function () {
  $('#isu-post-more-btn').on('click', function () {
    $.ajax({
      type: 'GET',
      url: '/posts',
      data: {
        max_created_at: $('.isu-post:last').attr('data-max')
      }
    }).done(function(data) {
      $('#isu-post-more').before(data);
    });
  });
});
