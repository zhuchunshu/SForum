<div class="row row-cards justify-content-center">
    <div class="col-md-12" id="topic">
        <div class="border-0 card">
            <div class="card-body topic">
                @if ($data->essence > 0)
                    <div class="ribbon bg-green text-h3">
                        {{__('app.essence')}}
                    </div>
                @endif
                <div class="row">
{{--                    标题--}}
                    <div class="col-md-12" id="title">
                        <h1 data-bs-toggle="tooltip" data-bs-placement="left" title="{{__('topic.title')}}">
                            @if ($data->topping > 0)
                                <span class="text-red">
                                    {{__('app.top')}}
                                </span>
                            @endif
                            {{ $data->title }}
                        </h1>
                    </div>

{{--                    面包屑--}}
                    <div class="col-md-12">
                        @include('App::topic.show.ol')
                    </div>
                    <hr class="hr-text" style="margin-top: 5px;margin-bottom: 5px">

{{--                    作者信息--}}
                    @include('App::topic.show.include.author')

{{--文章信息--}}
                    <article class="col-md-12 article" id="topic-content">
                        {!! ContentParse()->parse($data->post->content,$parseData) !!}
                    </article>

                </div>
            </div>

{{--            页脚--}}
            @include('App::topic.show.include.footer')

        </div>

    </div>

{{--    上下页--}}
    @include('App::topic.show.include.lfpage')
{{--    显示评论--}}
    @include('Comment::Widget.show-topic')
{{--    评论--}}
    @include('Comment::Widget.topic')

    @if(auth()->check())
{{--        举报模态--}}
        @include('App::topic.show.include.report')
    @endif
</div>

