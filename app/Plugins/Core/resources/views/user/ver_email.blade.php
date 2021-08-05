@extends("plugins.Core.app")
@section('title','验证邮箱')
@section('content')
    <div id="vue-user-my-ver-email">
        <div class="row row-cards">
            <div class="col-md-4">
                <form action="/user/ver_email?Redirect=/user/ver_email" method="post" enctype="multipart/form-data">
                    <x-csrf/>
                    <div class="card card-body">
                        <div class="card-title">基本设置</div>
                        <div class="mb-3">
                            <label class="form-label">用户名</label>
                            <div class="row g-2">
                                <div class="col">
                                    <input type="email" disabled class="form-control" value="{{auth()->data()->email}}">
                                </div>
                                <div class="col-auto">
                                    <button v-if="send" type="submit" name="send" value="send" class="btn btn-white btn-icon" aria-label="Button">
                                        发送验证码
                                    </button>
                                </div>
                            </div>
                        </div>
                        <div class="mb-3">
                            <label class="form-label">验证码</label>
                            <input type="text" name="captcha" class="form-control">
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