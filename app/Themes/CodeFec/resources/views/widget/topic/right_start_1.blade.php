{{--                标签信息--}}
<div class="col-md-12">
    <a href="/tags/{{$data->tag->id}}.html" class="card card-link text-primary-fg" style="background-color: {{$data->tag->color}}!important;">
        <div class="card-stamp">
            <div class="card-stamp-icon bg-yellow">
                <!-- Download SVG icon from http://tabler-icons.io/i/star -->
                {!! $data->tag->icon !!}
            </div>
        </div>
        <div class="card-body">
            <h3 class="card-title">{{$data->tag->name}}</h3>
            <p>{{ \Hyperf\Utils\Str::limit(core_default($data->tag->description, __("app.no description")), 32) }}</p>
        </div>
    </a>
</div>

{{--                作者信息--}}
<div class="col-md-12">
    <a class="card card-link" href="#">
        <div class="card-cover card-cover-blurred text-center" style="background-image: url({{get_user_settings(auth()->id(),'backgroundImg','/plugins/Core/image/user_background.jpg')}})">
            <span class="avatar avatar-xl avatar-thumb avatar-rounded" style="background-image: url({{super_avatar($data->user)}})"></span>
        </div>
        <div class="card-body text-center">
            <div class="card-title mb-1">{{$data->user->username}}</div>
            <div class="text-muted">本文作者，至今共发布{{$data->user->topic->count()}}篇文章</div>
        </div>
    </a>
</div>


