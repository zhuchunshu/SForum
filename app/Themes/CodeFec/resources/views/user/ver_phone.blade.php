@extends("App::app")
@section('title','验证手机')
@section('content')
    <div id="vue-user-my-ver-email">
        <div class="row row-cards justify-content-center">
            <div class="col-md-6">
                <form action="/user/ver_phone/send?Redirect=/user/ver_phone/send" method="post" enctype="multipart/form-data">
                    <x-csrf/>
                    <div class="card card-body">
                        <div class="card-title">手机验证</div>
                        <div class="mb-3">
                            <label class="form-label">手机号</label>
                            <input type="number" name="phone" class="form-control" minlength="11" maxlength="11" required>
                            <small>每天只有 {{get_options('core_user_sms_limit',1)}} 次免费发信机会，请认真填写手机号</small>
                        </div>
                        <div class="mb-3">
                            <button class="btn btn-primary" type="submit">发送验证码</button>
                        </div>
                    </div>
                </form>
            </div>

            <div class="col-md-6">
                <form action="/user/ver_phone?Redirect=/user/ver_phone" method="post" enctype="multipart/form-data">
                    <x-csrf/>
                    <div class="card card-body">
                        <div class="card-title">手机验证</div>
                        <div class="mb-3">
                            <label class="form-label">短信验证码</label>
                            <input type="text" name="code" class="form-control">
                        </div>
                        <div class="mb-3">
                            <button class="btn btn-primary" type="submit">提交</button>
                        </div>
                    </div>
                </form>
            </div>
        </div>
    </div>
@endsection

@section('scripts')
    <script src="{{mix("plugins/Core/js/user.js")}}"></script>
@endsection