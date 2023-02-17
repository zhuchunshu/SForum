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
                    @foreach((new \App\Plugins\User\src\Service\UserManagement())->get_all() as $item)
                        @include($item['edit_view'])
                    @endforeach
                    <div class="mb-3 mt-3">
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