<div class="col-md-12 card" x-data="auth">
    <div class="card-header">
        <div class="card-title">账号登陆设备信息</div>
        <div class="card-actions">
            <button @click="offlineAll()" class="btn btn-ghost-danger">下线除本机外所有设备</button>
        </div>
    </div>
    <div class="table-responsive">
        <table class="table card-table table-vcenter">
            <thead>
            <tr>
                <th>登陆时间</th>
                <th>Token</th>
                <th>浏览器</th>
                <th>操作系统</th>
                <th>登陆IP</th>
                <th colspan="2">IP归属地</th>
            </tr>
            </thead>
            <tbody>
            @foreach(\App\Plugins\User\src\Models\UsersAuth::query()->where('user_id',auth()->id())->orderByDesc('id')->get() as $auth)
                <tr>
                    <td class="text-muted">
                        {{$auth->created_at}}
                    </td>
                    <td class="text-muted">{{$auth->token}} @if(auth()->token()===$auth->token)
                            <span class="badge bg-red-lt">本机</span>
                        @endif </td>
                    <td class="text-muted">@if($auth->user_agent)
                            {!! \App\Plugins\User\src\Helpers\UserAgent::getBrowser($auth->user_agent) !!}
                        @else
                            {{"未知"}}
                        @endif</td>
                    <td class="text-muted">
                        @if($auth->user_agent)
                            {!! \App\Plugins\User\src\Helpers\UserAgent::getOs($auth->user_agent) !!}
                        @else
                            {{"未知"}}
                        @endif
                    </td>
                    <td class="text-muted">
                        @if($auth->user_ip)
                            {{\App\Plugins\User\src\Helpers\UserAgent::cutIp($auth->user_ip,3)}}
                        @else
                            {{"未知"}}
                        @endif
                    </td>
                    <td class="text-muted"
                        @if($auth->user_ip) x-text="await get_ip('{{\App\Plugins\User\src\Helpers\UserAgent::cutIp($auth->user_ip,3,'0')}}')" @endif>
                        @if($auth->user_ip)
                            加载中...
                        @else
                            {{"未知"}}
                        @endif
                    </td>
                    <td>
                        <button @click="offline('{{$auth->token}}')" class="btn btn-ghost-danger">下线</button>
                    </td>
                </tr>
            @endforeach
            </tbody>
        </table>
    </div>
</div>
<script src="{{file_hash("js/axios.min.js")}}"></script>
<script type="text/javascript" src="https://ip.useragentinfo.com/jsonp?ip="></script>
<script>
    document.addEventListener('alpine:init', () => {
        Alpine.data('auth', () => ({
            async get_ip(ip) {
                let r = await (await fetch('https://ip.useragentinfo.com/json?ip=' + ip)).json()

                return await r.province + " " + r.city + " " + r.area + " " + r.isp
            },
            offline(token) {
                axios.post("/api/user/setting/authOffline/check", {
                    _token: csrf_token,
                    token: token
                }).then(r => {
                    const data = r.data
                    if (data.success === false) {
                        swal('Error', data.result.msg, 'error')
                        return;
                    }
                    swal(data.result.msg, {
                        content: "input",
                        buttons: true
                    })
                        .then((value) => {
                            if (value !== null) {
                                if (!value) {
                                    swal(`验证码为空,拒绝下线设备请求`);
                                    return;
                                }
                                axios.post("/api/user/setting/authOffline/verify", {
                                    _token: csrf_token,
                                    token: token,
                                    code: value
                                }).then(r => {
                                    const result = r.data
                                    if (result.success === false) {
                                        swal('Error', result.result.msg, 'error')
                                        return;
                                    }
                                    swal('Success', result.result.msg, 'success')
                                    setTimeout(() => {
                                        location.reload()
                                    }, 1500)
                                })

                            }
                        });
                })
            },
            offlineAll() {
                axios.post("/api/user/setting/authOffline/all/check", {
                    _token: csrf_token,
                }).then(r => {
                    const data = r.data
                    if (data.success === false) {
                        swal('Error', data.result.msg, 'error')
                        return;
                    }
                    swal(data.result.msg, {
                        content: "input",
                        buttons: true
                    })
                        .then((value) => {
                            if (value !== null) {
                                if (!value) {
                                    swal(`验证码为空,拒绝下线设备请求`);
                                    return;
                                }
                                axios.post("/api/user/setting/authOffline/all/verify", {
                                    _token: csrf_token,
                                    code: value
                                }).then(r => {
                                    const result = r.data
                                    if (result.success === false) {
                                        swal('Error', result.result.msg, 'error')
                                        return;
                                    }
                                    swal('Success', result.result.msg, 'success')
                                    setTimeout(() => {
                                        location.reload()
                                    }, 1500)
                                })

                            }
                        });
                })
            }
        }))
    })
</script>