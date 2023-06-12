<div core-show="comment" comment-id="{{$value->id}}"
     class="col-md-12 markdown mt-3 mb-2 px-3" style="font-size: 15px">
    @if($value->parent_id)
        @if(@$value->parent->id)
            <div class="quote">
                <blockquote>
                    <a style="font-size:13px;" href="{{$value->parent_url}}"
                       target="_blank">
                        <span style="color:#999999">{{$value->parent->user->username}} {{__("app.Published on")}} {{format_date($value->parent->created_at)}}</span>
                    </a>
                    <br>
                    {!! \Hyperf\Utils\Str::limit(remove_bbCode(strip_tags($value->parent->post->content)),60) !!}
                </blockquote>
            </div>
        @else
            <div class="quote">
                <blockquote>
                    引用的评论已被删除
                </blockquote>
            </div>
        @endif
    @endif
    {!!CommentContentParse()->parse($value->post->content,['comment' => $value,'topic' => $data]) !!}
</div>