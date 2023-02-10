<div class="border-0 card card-body">
    <div class="row">
        <div class="col-md-12">
            <div class="row">
                <div class="col-auto">
                    <span class="avatar" style="background-image: url({{super_avatar(auth()->data())}})"></span>
                </div>
                <div class="col text-truncate">
                    <a style="white-space:nowrap;" href="/users/{{auth()->id()}}.html" class="text-body text-truncate">{{auth()->data()->username}}</a>
                    <br>
                    <small data-bs-toggle="tooltip" data-bs-placement="top" data-bs-original-title="{{$data->created_at}}" class="text-muted text-truncate mt-n1">
                        {{__("app.Published on")}}:{{$data->created_at}}
                    </small>
                </div>
            </div>
        </div>
        <div class="col-md-12">
            <div class="hr-text" style="margin-bottom:5px;margin-top:15px">回复内容</div>
        </div>
        <div class="col-md-12 markdown">
            <div class="quote">
                <blockquote>
                    <a style="font-size:13px;" href="{{$data->parent_url}}" target="_blank">
                        <span style="color:#999999" >{{$data->parent->user->username}} {{__("app.Published on")}} {{$data->created_at}}</span>
                    </a>
                    <br>
                    {{\Hyperf\Utils\Str::limit(remove_bbCode(strip_tags($data->parent->post->content)),60)}}
                </blockquote>
            </div>
            {{\Hyperf\Utils\Str::limit(remove_bbCode(strip_tags($data->post->content)),100)}}
        </div>
    </div>
</div>