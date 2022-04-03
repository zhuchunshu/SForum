@extends("app")
@section('title','创建邀请码')
@section('content')
    <div class="col-md-12">
        <div class="border-0 card">
            <div class="card-body">
                <h3 class="card-title">创建邀请码</h3>
                <form method="post" action="">
                    <x-csrf/>
                    <div class="row row-cards">
                        <div class="col-md-4">
                            <label for="" class="form-label">前缀</label>
                            <input type="text" class="form-control" name="before" placeholder="可以为空">
                        </div>
                        <div class="col-md-4">
                            <label for="" class="form-label">后缀</label>
                            <input type="text" class="form-control" name="after" placeholder="可以为空">
                        </div>
                        <div class="col-md-4">
                            <label for="" class="form-label">生成数量</label>
                            <input type="number" class="form-control" name="count" placeholder="建议不要一次生成太多">
                        </div>
                        <div class="col-md-4">
                            <button class="btn btn-light">生成</button>
                        </div>
                    </div>
                </form>
            </div>
        </div>
    </div>
@endsection