<div class="row row-cards justify-content-center">
    <div class="col-md-6">
        <div class="card">
            <div class="card-body text-center">
                <div class="mb-3">
                    <span class="avatar avatar-xl avatar-rounded" style="background-image: url({{super_avatar($data)}})"></span>
                </div>
                <div class="card-title mb-1"><a href="/users/{{$data->username}}.html">{{$data->username}}</a></div>
                <div class="text-muted">共{{$data->fans}}位粉丝</div>
            </div>
            <a user-click="user_follow" user-id="{{$data->id}}" class="card-btn">关注</a>
        </div>
    </div>
</div>