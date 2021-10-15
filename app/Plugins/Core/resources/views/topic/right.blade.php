
{{--                作者信息--}}
<div class="col-md-10">
    <div class="border-0 card">
        <div class="card-body">
            <div class="row">
                <div class="col-md-12">
                    <div class="row justify-content-center">
                        <span class="avatar avatar-lg center-block" style="background-image: url({{super_avatar($data->user)}})"></span>
                        <br>
                        <b class="card-title text-h2 text-center" style="margin-top: 5px;margin-bottom:2px">{{ $data->user->username }}</b>
                        <span class="text-center" style="color:rgba(0,0,0,.45)">共 {{$data->user->fans}} 位粉丝</span>
                        <br>
                    </div>
                </div>
            </div>
        </div>
        <div class="d-flex">
            <a class="card-btn cursor-pointer" user-click="user_follow" user-id="{{ $data->user_id }}">
                <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-user-plus" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                    <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                    <circle cx="9" cy="7" r="4"></circle>
                    <path d="M3 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2"></path>
                    <path d="M16 11h6m-3 -3v6"></path>
                </svg>
                <span>关注</span></a>
            <a href="/users/{{$data->user->username}}.html" class="card-btn">查看</a>
        </div>
    </div>
</div>

{{--                标签信息--}}
<div class="col-md-10">
    <div class="border-0 card">
        <div class="card-body">
            <div class="row">
                <div class="col-md-12">
                   <div class="row justify-content-center">
                       <span class="avatar avatar-lg center-block" style="background-image: url({{$data->tag->icon}})"></span>
                       <br>
                       <b class="card-title text-h2 text-center" style="margin-top: 5px;margin-bottom:2px">{{ $data->tag->name }}</b>
                       <span class="text-center" style="color:rgba(0,0,0,.45)">{{ \Hyperf\Utils\Str::limit(core_default($data->tag->description, '暂无描述'), 32) }}</span>
                       <br>
                       <a href="/tags/{{ $data->tag->id }}.html" class="btn btn-azure text-center">查看此标签</a>
                   </div>
                </div>
            </div>
        </div>
    </div>
</div>
