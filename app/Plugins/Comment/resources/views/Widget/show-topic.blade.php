@if($comment_count)
    <div class="col-md-12"  comment-load="topic" topic-id="{{$data->id}}">
        <div class="row row-cards">
            <span class="text-center" comment-load="remove"><h1>正在加载评论<span class="animated-dots"></span></h1></span>
        </div>
    </div>
@endif
