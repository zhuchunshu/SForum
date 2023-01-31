@extends("app")
@section('title','邮箱设置')

@section('content')

    <div class="col-md-12">
        <div class="row row-cards">
            <div class="col-md-6">
                <div class="card">
                    <div class="card-body">
                        <h3 class="card-title">测试发信</h3>
                        <form action="" method="post">
                            <x-csrf/>
                            <div class="mb-3">
                                <label for="" class="form-label">邮箱</label>
                                <input type="email" class="form-control" name="email" placeholder="邮箱地址">
                            </div>
                            <div class="mb-3">
                                <button class="btn btn-primary">提交</button>
                            </div>
                        </form>
                    </div>
                </div>
            </div>
        </div>
    </div>

@endsection