@extends("App::app")

@section('title','支付验证')

@section('content')
    <div class="page page-center">
        <div class="container container-tight py-4">
            <div class="text-center mb-4">
                <a href="." class="navbar-brand navbar-brand-autodark"><img src="./static/logo.svg" height="36" alt=""></a>
            </div>
            <form class="card card-md" action="" method="post" autocomplete="off" novalidate>
                <x-csrf/>()
                <div class="card-body text-center">
                    <div class="mb-3">
                        <h2 class="card-title">支付锁</h2>
                        <p class="text-muted">你即将使用账户余额付款，安全起见，需要验证你的密码</p>
                    </div>
                    <div class="mb-4">
                        <p>关联订单: <a href="/user/order/{{$order->id}}.order">{{$order->id}}</a></p>
                       付款金额: <b class="text-red">{{$order->amount}}</b> 付款前余额: <b class="text-red">{{$money}}</b> 付款后余额: <b class="text-red">{{$money-$order->amount}}</b>
                    </div>
                    <div class="mb-4">
                        <span class="avatar avatar-xl mb-3" style="background-image: url({{super_avatar(auth()->data())}})"></span>
                        <h3>{{auth()->data()->username}}</h3>
                    </div>
                    <div class="mb-4">
                        <input type="password" name="password" class="form-control" placeholder="Password&hellip;">
                    </div>
                    <div>
                        <button class="btn btn-primary w-100">
                            <!-- Download SVG icon from http://tabler-icons.io/i/lock-open -->
                            <svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><rect x="5" y="11" width="14" height="10" rx="2" /><circle cx="12" cy="16" r="1" /><path d="M8 11v-5a4 4 0 0 1 8 0" /></svg>
                            验证并支付
                        </button>
                    </div>
                </div>
            </form>
        </div>
    </div>

@endsection