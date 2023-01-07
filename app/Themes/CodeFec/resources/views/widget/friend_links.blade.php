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
            <ul style="padding-left: 0;list-style: none;">
                @foreach(\App\Plugins\Core\src\Models\FriendLink::query()->orderByDesc('to_sort')->get(['name','link','to_sort','_blank']) as $data)
                    <li class="mb-1" style="float: left;list-style: outside none none;padding: 3px;line-height: 1.6">
                        <a class="text-muted h4" @if((int)$data->_blank===1) target="_blank" @endif href="{{$data->link}}">{{$data->name}}</a>
                    </li>
                @endforeach
            </ul>
        </div>
    </div>
</div>