<div class="col-md-12">
    <div class="card">
        <div class="card-status-top bg-danger"></div>
        <div class="card-header">
            <h3 class="card-title">友情链接</h3>
            @if(get_options('theme_common_friend_links_apply'))
                <div class="card-actions">
                    <a href="{{get_options('theme_common_friend_links_apply')}}">申请友链</a>
                </div>
            @endif
        </div>
        <div class="card-body">
            @foreach(\App\Plugins\Core\src\Models\FriendLink::query()->orderByDesc('to_sort')->get(['name','link','to_sort']) as $data)
                <span class="antialiased"><a class="text-muted h3" href="{{$data->link}}">{{$data->name}}</a></span>
            @endforeach
        </div>
    </div>
</div>