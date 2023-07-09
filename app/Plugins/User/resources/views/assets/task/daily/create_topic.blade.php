{{--获取今天发帖获取的奖励次数--}}
@php($today_topic_count = \App\Plugins\User\src\Models\UsersAward::where('user_id',auth()->id())->where('name','create_topic')->whereDate('created_at',\Carbon\Carbon::today())->count())
@php($task_create_toppic=$today_topic_count>=(int)get_hook_credit_options('create_topic_award_number',10))
<div class="row">
    <div class="col-auto">
                                <span class="avatar @if($task_create_toppic) text-primary @else text-warning @endif"
                                      style="background: unset">
                                    <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-pencil" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M4 20h4l10.5 -10.5a1.5 1.5 0 0 0 -4 -4l-10.5 10.5v4"></path>
   <path d="M13.5 6.5l4 4"></path>
</svg>
                                </span>
    </div>
    <div class="col text-truncate">
        <a class="text-body d-block">{{$daily['name']}}</a>
        <div class="text-muted text-truncate mt-n1">当前已完成: {{$today_topic_count}}/{{(int)get_hook_credit_options('create_topic_award_number',10)}}</div>
    </div>
    @if(!$task_create_toppic)
        <div class="col-auto">
            <a href="/topic/create" target="_blank" class="btn btn-link">去完成</a>
        </div>
    @endif
</div>