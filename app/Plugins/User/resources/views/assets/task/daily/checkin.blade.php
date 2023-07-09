{{--            判断用户今天是否已经签到--}}
@if(\App\Plugins\User\src\Models\UsersAward::where('user_id',auth()->id())->where('name','checkin')->whereDate('created_at',\Carbon\Carbon::today())->exists())
    @php($checkin = true)
@else
    @php($checkin = false)
@endif

<div class="row" x-data="user_task_checkin">
    <div class="col-auto">
                                <span class="avatar @if($checkin) text-primary @else text-warning @endif"
                                      style="background: unset">
                                    @if(!$checkin)
                                        <svg xmlns="http://www.w3.org/2000/svg"
                                             class="icon icon-tabler icon-tabler-square-rounded" width="24" height="24"
                                             viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                                             stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M12 3c7.2 0 9 1.8 9 9s-1.8 9 -9 9s-9 -1.8 -9 -9s1.8 -9 9 -9z"></path>
</svg>
                                    @else
                                        <svg xmlns="http://www.w3.org/2000/svg"
                                             class="icon icon-tabler icon-tabler-checkbox" width="24" height="24"
                                             viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                                             stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M9 11l3 3l8 -8"></path>
   <path d="M20 12v6a2 2 0 0 1 -2 2h-12a2 2 0 0 1 -2 -2v-12a2 2 0 0 1 2 -2h9"></path>
</svg>
                                    @endif
                                </span>
    </div>
    <div class="col text-truncate">
        <a class="text-body d-block">{{$daily['name']}}</a>
        <div class="text-muted text-truncate mt-n1">
            @if($checkin)
                {{__('user.already_checkin_tips')}}
            @else
                {{__('user.not_checkin_tips')}}
            @endif
        </div>
    </div>

    @if(!$checkin)
        <div class="col-auto">
            <a @@click="checkin" class="btn btn-link">立即签到</a>
        </div>
    @endif
</div>
<script>
    document.addEventListener('alpine:init', () => {
        Alpine.data('user_task_checkin', () => ({
            checkin() {
                fetch("/api/user/task/checkin", {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Accept': 'application/json',
                    },
                    body: JSON.stringify({
                        _token: csrf_token
                    })
                }).then(res => res.json()).then(res => {
                    if (res.success) {
                        swal("签到成功", res.result.msg, "success");
                        setTimeout(() => {
                            location.reload()
                        }, 1500)
                    } else {
                        swal("签到失败", res.result.msg, "error")
                    }
                })
            }
        }))
    })
</script>