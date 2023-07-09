{{--            判断用户今天是否已经签到--}}
@if(\App\Plugins\User\src\Models\UsersAward::where('user_id',auth()->id())->where('name','set_avatar')->exists())
    @php($avatar = true)
@else
    @php($avatar = false)
@endif

<div class="row">
    <div class="col-auto">
                                <span class="avatar @if($avatar) text-primary @else text-warning @endif"
                                      style="background: unset">
                                    <svg xmlns="http://www.w3.org/2000/svg"
                                         class="icon icon-tabler icon-tabler-user-heart" width="24" height="24"
                                         viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                                         stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M8 7a4 4 0 1 0 8 0a4 4 0 0 0 -8 0"></path>
   <path d="M6 21v-2a4 4 0 0 1 4 -4h.5"></path>
   <path d="M18 22l3.35 -3.284a2.143 2.143 0 0 0 .005 -3.071a2.242 2.242 0 0 0 -3.129 -.006l-.224 .22l-.223 -.22a2.242 2.242 0 0 0 -3.128 -.006a2.143 2.143 0 0 0 -.006 3.071l3.355 3.296z"></path>
</svg>
                                </span>
    </div>
    <div class="col text-truncate">
        <a class="text-body d-block">{{$system['name']}}</a>
        <div class="text-muted text-truncate mt-n1">
            @if($avatar)
                {{__('user.task.set_avatar_done')}}
            @else
                {{__('user.task.set_avatar_undone')}}
            @endif
        </div>
    </div>
    @if(!$avatar)
        <div class="col-auto">
            <a href="/user/setting" class="btn btn-link">立即上传</a>
        </div>
    @else
        <div class="col-auto">
            <span class="status status-green">
                        已完成
            </span>
        </div>
    @endif
</div>