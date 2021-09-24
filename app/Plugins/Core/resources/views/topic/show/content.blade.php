<div class="row row-cards justify-content-center">
    <div class="col-md-12">
        <div class="border-0 card card-body topic">
            @if($data->essence>0)
                <div class="ribbon bg-green text-h3">
                    精华
                </div>
            @endif
            <div class="row">
                <div class="col-md-12" id="title">
                    <h1 data-bs-toggle="tooltip" data-bs-placement="top" title="帖子标题">

                        @if($data->topping>0)
                            <span class="text-red">
                                置顶
                            </span>
                        @endif

                        {{$data->title}}
                    </h1>
                </div>
                <div class="col-md-12">
                    @include('plugins.Core.topic.show.ol')
                </div>
                <hr class="hr-text" style="margin-top: 5px;margin-bottom: 5px">
                <div class="col-md-12" id="author">
                    <div class="row">
                        <div class="col-auto">
                            <a class="avatar" href="/users/{{$data->user->username}}.html" style="background-image: url({{super_avatar($data->user)}})"></a>
                        </div>
                        <div class="col">
                            <div class="topic-author-name">
                                <a href="/users/{{$data->user->username}}.html" class="text-reset">{{$data->user->username}}</a>
                            </div>
                            <div>发表于:{{format_date($data->created_at)}}</div>
                        </div>
                    </div>
                </div>

                <div class="col-md-12 vditor-reset" id="topic-content">
                    {!! ShortCodeR()->handle($data->content) !!}
                </div>

            </div>

        </div>
    </div>

</div>