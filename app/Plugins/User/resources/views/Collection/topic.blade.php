<div class="border-0 card card-body">
    <div class="row">
        <div class="col-md-12">
            <div class="row">
                <div class="col-auto">
                    <span class="avatar" style="background-image: url({{super_avatar($user_data)}})"></span>
                </div>
                <div class="col text-truncate">
                    <a style="white-space:nowrap;" href="/users/{{$topic->user->username}}.html" class="text-body text-truncate">{{$topic->user->username}}</a>
                    <br>
                    <small data-bs-toggle="tooltip" data-bs-placement="top" data-bs-original-title="{{$topic->created_at}}" class="text-muted text-truncate mt-n1">发表于:{{$topic->created_at}}</small>
                </div>
            </div>
        </div>
        <div class="col-md-12">
            <div class="hr-text" style="margin-bottom:5px;margin-top:15px">帖子摘要</div>
        </div>
        <div class="col-md-12 markdown vditor-reset">
            {{\Hyperf\Utils\Str::limit(core_default(deOptions($topic->options)["summary"],"未捕获到本文摘要"),300)}}
        </div>
    </div>
</div>