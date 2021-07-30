@extends("plugins.Core.app")
@section('title','个人设置')
@section('content')
    <div id="vue-user-my-setting">
        <form action="/user/myUpdate?Redirect=/user/setting" method="post" enctype="multipart/form-data">
            <div class="row row-cards">
                <div class="col-md-4">
                    <x-csrf/>
                    <div class="card card-body">
                        <div class="card-title">基本设置</div>
                        <div class="mb-3">
                            <label class="form-label">用户名</label>
                            <input disabled type="text" v-model="username" class="form-control">
                        </div>
                        <div class="mb-3">
                            <label class="form-label">邮箱</label>
                            <input disabled type="email" v-model="email" class="form-control">
                        </div>
                        <div class="mb-3">
                            <label class="form-label">旧密码</label>
                            <input type="text" v-model="old_pwd" name="old_pwd" class="form-control">
                        </div>
                        <div class="mb-3">
                            <label class="form-label">新密码</label>
                            <input type="text" v-model="new_pwd" name="new_pwd" class="form-control">
                        </div>

                    </div>
                </div>
                <div class="col-md-12">
                    <button class="btn btn-primary" type="submit">提交</button>
                </div>
            </div>
        </form>
    </div>
@endsection

@section('scripts')
    <script src="{{mix("plugins/Core/js/user.js")}}"></script>
@endsection