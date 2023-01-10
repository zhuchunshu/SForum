@extends("app")

@section('title',"文档设置")


@section('content')
    <div class="col-md-12">
        <div class="border-0 card">
            <div class="card-body">
                <h3 class="card-title">文档设置</h3>
                请前往用户组对指定用户组设置文档权限
            </div>
        </div>
    </div>
@endsection

@section('scripts')
    <script src="{{ mix("plugins/User/js/user.js") }}"></script>
@endsection