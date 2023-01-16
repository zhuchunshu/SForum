@extends("app")

@section('title',"用户信息")


@section('content')
    <div class="col-md-12">
        <div class="card">
            <div class="card-header">
                <h3 class="card-title">【{{$user->username}}】信息</h3>
                <div class="card-actions">
                    <a href="/admin/users/{{$user->id}}/show" class="btn">查看</a>
                </div>
            </div>
            <div class="card-body">
                <form action="/admin/users/{{$user->id}}/edit" method="post">
                    <x-csrf/>()
                    <div class="mb-3">
                        <label for="" class="form-label">用户名</label>
                        <input class="form-control" type="text"  name="username" value="{{$user->username}}">
                    </div>

                    <div class="mb-3">
                        <label for="" class="form-label">邮箱</label>
                        <input class="form-control" type="email" name="email" value="{{$user->email}}">
                    </div>


                    <div class="mb-3">
                        <label for="" class="form-label">手机号</label>
                        <input class="form-control" type="text" name="phone" value="@if($user->phone){{$user->phone}}@else 暂无 @endif">
                    </div>

                    <div class="mb-3">
                        <label class="form-label">
                            用户组
                        </label>
                        <select name="class_id" class="form-select">
                            @foreach(App\Plugins\User\src\Models\UserClass::query()->get() as $value)
                                <option value="{{$value->id}}" @if ($value->id===$user->Class->id) selected @endif>{{$value->name}}</option>
                            @endforeach
                        </select>
                    </div>

                    <div class="mb-3">
                        <label for="" class="form-label">手机验证时间</label>
                        <input class="form-control" type="text" name="phone_ver_time" value="@if($user->phone_ver_time) {{$user->phone_ver_time}} @else 未验证 @endif">
                    </div>


                    <div class="mb-3">
                        <label for="" class="form-label">签名</label>
                        <textarea rows="4" name="options[qianming]" class="form-control">{{$user->options->qianming}}</textarea>
                    </div>

                    <div class="mb-3">
                        <label for="" class="form-label">qq</label>
                        <input class="form-control" type="text" name="options[qq]" value="@if($user->options->qq) {{$user->options->qq}} @endif">
                    </div>
                    <div class="mb-3">
                        <label for="" class="form-label">微信</label>
                        <input class="form-control" type="text" name="options[wx]" value="@if($user->options->wx) {{$user->options->wx}} @endif">
                    </div>


                    <div class="mb-3">
                        <label for="" class="form-label">网站</label>
                        <input class="form-control" type="text" name="options[website]" value="@if($user->options->website) {{$user->options->website}}  @endif">
                    </div>

                    <div class="mb-3">
                        <label for="" class="form-label">对外邮箱</label>
                        <input class="form-control" type="text" name="options[email]" value="@if($user->options->email) {{$user->options->email}} @endif">
                    </div>

                    <div class="mb-3">
                        <label for="" class="form-label">积分</label>
                        <input class="form-control" type="text" name="options[credits]" value="@if($user->options->credits) {{$user->options->credits}} @endif">
                    </div>

                    <div class="mb-3">
                        <label for="" class="form-label">金币</label>
                        <input class="form-control" type="text" name="options[golds]" value="@if($user->options->golds) {{$user->options->golds}} @endif">
                    </div>

                    <div class="mb-3">
                        <label for="" class="form-label">经验</label>
                        <input class="form-control" type="text" name="options[exp]" value="@if($user->options->exp) {{$user->options->exp}} @endif">
                    </div>

                    <div class="mb-3">
                        <label for="" class="form-label">余额</label>
                        <input class="form-control" type="text" name="options[money]" value="@if($user->options->money) {{$user->options->money}} @endif">
                    </div>

                    <div class="mb-3">
                        <button class="btn btn-primary">提交</button>
                    </div>
                </form>
            </div>
        </div>
    </div>
@endsection

@section('scripts')
    <script src="{{ mix("plugins/User/js/user.js") }}"></script>
@endsection