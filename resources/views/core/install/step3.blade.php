@extends('core.install')
@section('title', '第三步')
@section('content')
    <div class="mb-3">
        <label class="form-label">数据库地址</label>
        <input name="DB_HOST" value="{{ env('DB_HOST') }}" type="text" class="form-control" autocomplete="off"
            required>
    </div>
    <div class="mb-3">
        <label class="form-label">数据库名</label>
        <input name="DB_DATABASE" value="{{ env('DB_DATABASE') }}" type="text" class="form-control" autocomplete="off"
            required>
    </div>
    <div class="mb-3">
        <label class="form-label">数据库用户名</label>
        <input name="DB_USERNAME" value="{{ env('DB_USERNAME') }}" type="text" class="form-control" autocomplete="off"
            required>
    </div>
    <div class="mb-3">
        <label class="form-label">数据库密码</label>
        <input name="DB_PASSWORD" value="{{ env('DB_PASSWORD') }}" type="password" class="form-control" autocomplete="off"
            required>
    </div>

@endsection

@section('footer')
    <div class="col-4">
        <div class="progress">
            <div class="progress-bar" style="width: 60%" role="progressbar" aria-valuenow="60" aria-valuemin="0"
                aria-valuemax="100">
            </div>
        </div>
        <span>60% Complete</span>
    </div>
    <div class="col">
        <div class="btn-list justify-content-end">
            <a href="/install?step=2" class="btn btn-link link-secondary">
                上一步
            </a>
            <button type="submit" class="btn btn-primary">
                下一步
            </button>
        </div>
    </div>
@endsection

@section('action', '/install?step=3')

@section('hr', '数据库连接配置')

@section('img','/install/step3.svg')