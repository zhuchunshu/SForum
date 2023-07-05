<div class="col-md-12">
    <div class="card">
        <div class="card-header" style="padding: 10px 20px 10px 10px">
            <h3 class="card-title">友情链接</h3>
            @if(get_options('theme_common_friend_links_apply'))
                <div class="card-actions">
                    <a href="{{get_options('theme_common_friend_links_apply')}}">申请友链</a>
                </div>
            @endif
        </div>
        <div class="card-body">
            <ul style="padding-left: 0;list-style: none;">
                @foreach(\App\Plugins\Core\src\Models\FriendLink::query()->orderByDesc('to_sort')->where('hidden',false)->get(['name','link','to_sort','_blank']) as $data)
                    <li class="mb-1" style="float: left;list-style: outside none none;padding: 3px;line-height: 1.6">
                        <a class="text-muted h4" @if((int)$data->_blank===1) target="_blank" @endif href="{{$data->link}}">{{$data->name}}</a>
                    </li>
                @endforeach
            </ul>
        </div>
        @if(get_options('theme_common_friend_links_all'))
        <div class="card-footer" style="padding: 10px 20px 10px 10px">
            <div class="row align-items-center">
                <div class="col-auto">
                    <a href="{{get_options('theme_common_friend_links_all')}}">查看全部</a>
                </div>
            </div>
        </div>
        @endif
    </div>
</div>