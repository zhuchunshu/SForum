{{--获取今天发帖获取的奖励次数--}}
@php($today_topic_comment_count = \App\Plugins\User\src\Models\UsersAward::where('user_id',auth()->id())->where('name','create_topic_comment')->whereDate('created_at',\Carbon\Carbon::today())->count())
@php($task_create_toppic_comment=$today_topic_comment_count>=(int)get_hook_credit_options('create_topic_comment_award_number',20))
<div class="row">
    <div class="col-auto">
                                <span class="avatar @if($task_create_toppic_comment) text-primary @else text-warning @endif"
                                      style="background: unset">
                                    <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-message-plus" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M8 9h8"></path>
   <path d="M8 13h6"></path>
   <path d="M12.01 18.594l-4.01 2.406v-3h-2a3 3 0 0 1 -3 -3v-8a3 3 0 0 1 3 -3h12a3 3 0 0 1 3 3v5.5"></path>
   <path d="M16 19h6"></path>
   <path d="M19 16v6"></path>
</svg>
                                </span>
    </div>
    <div class="col text-truncate">
        <a class="text-body d-block">{{$daily['name']}}</a>
        <div class="text-muted text-truncate mt-n1">当前已完成: {{$today_topic_comment_count}}/{{(int)get_hook_credit_options('create_topic_comment_award_number',10)}}</div>
    </div>
    @if(!$task_create_toppic_comment)
        <div class="col-auto">
            <a href="/{{\App\Plugins\Topic\src\Models\Topic::inRandomOrder()->select('id','created_at')->whereBetween('created_at',[\Carbon\Carbon::now()->subMonth(),\Carbon\Carbon::now()])->first()->id}}.html#topic-comment" target="_blank" class="btn btn-link">去完成</a>
        </div>
    @endif
</div>