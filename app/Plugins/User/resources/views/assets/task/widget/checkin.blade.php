<div class="col-12">
    <div class="card">
        <div class="card-header" style="padding: 10px 20px 10px 10px">
            <h3 class="card-title">签到</h3>
            <div class="card-actions">
                <a href="/users/{{auth()->id()}}.html?m=users_home_menu_12">任务中心</a>
            </div>
        </div>
        <div class="card-body text-center" style="font-size: 18px">
            @if (get_hook_credit_options('checkin_check', 'true') !== 'true')
                签到功能已关闭，前往<a href="/users/{{auth()->id()}}.html?m=users_home_menu_12">任务中心</a>可赚取更多资产
            @else
                @if(\App\Plugins\User\src\Models\UsersAward::where('user_id', auth()->id())->where('name', 'checkin')->whereDate('created_at', \Carbon\Carbon::today())->exists())
                今日已签到，前往<a href="/users/{{auth()->id()}}.html?m=users_home_menu_12">任务中心</a>可赚取更多资产
                @else
                    <button id="home-right-widget-task-checkin-button" class="btn btn-primary">立即签到</button>
                @endif
            @endif
        </div>
    </div>
</div>
<script>

    const btn = document.getElementById("home-right-widget-task-checkin-button")
    if(btn){
        btn.addEventListener('click', function () {
            // 使用fetch发送post请求
            fetch('/api/user/task/checkin', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    _token:csrf_token
                })
            }).then(function (response) {
                // 处理响应数据
                response.json().then(function (data) {
                    if(data.success===true){
                        swal("签到成功", data.result.msg, "success");
                        setTimeout(()=>{
                            location.reload()
                        },1500)
                    }else{
                        swal("签到失败", data.result.msg, "error");
                    }
                })
            })
        })
    }

</script>