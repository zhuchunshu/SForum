@extends('core.install')
@section('title','第五步')
@section('content')
    <div class="mb-3">
        <label class="form-label">管理员邮箱</label>
        <input name="email" type="email" class="form-control" autocomplete="off" required>
    </div>
    <div class="mb-3">
        <label class="form-label">管理员用户名</label>
        <input name="username" type="text" class="form-control" autocomplete="off" required>
    </div>
    <div class="mb-3">
        <label class="form-label">管理员密码</label>
        <input name="password" type="password" class="form-control" autocomplete="off" required>
    </div>

@endsection

@section('footer')
    <div class="col-4">
        <div class="progress">
            <div class="progress-bar" style="width: 90%" role="progressbar" aria-valuenow="90"
                 aria-valuemin="0" aria-valuemax="100">
            </div>
        </div>
        <span>90% Complete</span>
    </div>
    <div class="col">
        <div class="btn-list justify-content-end">
            <a href="/install?step=4" class="btn btn-link link-secondary">
                上一步
            </a>
            <button type="submit" class="btn btn-primary">
                下一步
            </button>
        </div>
    </div>
@endsection

@section('action', '/install?step=5')

@section('hr','注册管理员账号')
@section('img','/install/step5.svg')