@extends("app")

@section('title',"用户信息")


@section('content')
    <div class="col-md-12">
        <div class="card">
            <div class="card-header">
                <h3 class="card-title">【{{$user->username}}】信息</h3>
                <div class="card-actions">
                    <a href="/admin/users/{{$user->id}}/edit" class="btn">修改</a>
                </div>
            </div>
            <div class="card-body">
                <div class="mb-3">
                    <label for="" class="form-label">用户名</label>
                    <input class="form-control" type="text" disabled value="{{$user->username}}">
                </div>

                <div class="mb-3">
                    <label for="" class="form-label">邮箱</label>
                    <input class="form-control" type="email" disabled value="{{$user->email}}">
                </div>

                <div class="mb-3">
                    <label for="" class="form-label">邮箱验证时间</label>
                    <input class="form-control" type="text" disabled value="@if($user->email_ver_time) {{$user->email_ver_time}} @else 未验证 @endif">
                </div>



                <div class="mb-3">
                    <label for="" class="form-label">手机号</label>
                    <input class="form-control" type="text" disabled value="@if($user->phone){{$user->phone}}@else 暂无 @endif">
                </div>

                <div class="mb-3">
                    <label for="" class="form-label">用户组</label>
                    <input class="form-control" type="text" disabled value="{{$user->Class->name}}">
                </div>

                <div class="mb-3">
                    <label for="" class="form-label">手机验证时间</label>
                    <input class="form-control" type="text" disabled value="@if($user->phone_ver_time) {{$user->phone_ver_time}} @else 未验证 @endif">
                </div>

                <div class="mb-3">
                    <label for="" class="form-label">注册时间</label>
                    <input class="form-control" type="text" disabled value="@if($user->created_at) {{$user->created_at}} @else 暂无 @endif">
                </div>

                <div class="mb-3">
                    <label for="" class="form-label">签名</label>
                    <textarea rows="4" class="form-control" disabled>{{$user->Options->qianming}}</textarea>
                </div>

                <div class="mb-3">
                    <label for="" class="form-label">qq</label>
                    <input class="form-control" type="text" disabled value="@if($user->Options->qq) {{$user->Options->qq}} @else 暂无 @endif">
                </div>
                <div class="mb-3">
                    <label for="" class="form-label">微信</label>
                    <input class="form-control" type="text" disabled value="@if($user->Options->wx) {{$user->Options->wx}} @else 暂无 @endif">
                </div>


                <div class="mb-3">
                    <label for="" class="form-label">网站</label>
                    <input class="form-control" type="text" disabled value="@if($user->Options->website) {{$user->Options->website}} @else 暂无 @endif">
                </div>

                <div class="mb-3">
                    <label for="" class="form-label">对外邮箱</label>
                    <input class="form-control" type="text" disabled value="@if($user->Options->email) {{$user->Options->email}} @else 暂无 @endif">
                </div>

                <div class="mb-3">
                    <label for="" class="form-label">积分</label>
                    <input class="form-control" type="text" disabled value="@if($user->Options->credits) {{$user->Options->credits}} @else 暂无 @endif">
                </div>

                <div class="mb-3">
                    <label for="" class="form-label">金币</label>
                    <input class="form-control" type="text" disabled value="@if($user->Options->golds) {{$user->Options->golds}} @else 暂无 @endif">
                </div>

                <div class="mb-3">
                    <label for="" class="form-label">经验</label>
                    <input class="form-control" type="text" disabled value="@if($user->Options->exp) {{$user->Options->exp}} @else 暂无 @endif">
                </div>

                <div class="mb-3">
                    <label for="" class="form-label">余额</label>
                    <input class="form-control" type="text" disabled value="@if($user->Options->money) {{$user->Options->money}} @else 暂无 @endif">
                </div>
            </div>
        </div>
    </div>
@endsection

@section('scripts')
    <script src="{{ mix("plugins/User/js/user.js") }}"></script>
@endsection