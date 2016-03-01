$(function () {
    $('#isu-post-more-btn').on('click', function () {
        $.get('/posts',
            { max_created_at: $('.isu-post:last').data('max') },
            function( data ) { $('#isu-post-more').before(data) }
        );
    });
});
