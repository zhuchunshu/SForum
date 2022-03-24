<div class="border-0 card card-body">
    <div class="row">
        <div class="col-md-12">
            <div class="row">
                <div class="col-auto">
                    <span class="avatar" style="background-image: url({{super_avatar($user_data)}})"></span>
                </div>
                <div class="col text-truncate">
                    <a style="white-space:nowrap;" href="/users/{{$user_data->username}}.html" class="text-body text-truncate">{{$user_data->username}}</a>
                    <br>
                    赞了你的帖子: <a href="/{{$data->id}}.html">{{$data->title}}</a>
                </div>
            </div>
        </div>
    </div>
</div>