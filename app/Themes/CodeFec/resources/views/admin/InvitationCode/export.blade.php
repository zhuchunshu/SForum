@extends("app")
@section('title','导出邀请码')
@section('content')
    <div class="col-md-12" id="vue-admin-invitationCode-export">
        <div class="border-0 card">
            <div class="card-body">
                <div class="col">
                    <h3 class="card-title">导出未使用邀请码</h3>
                </div>
                <form action="" method="post">
                    <x-csrf/>
                    <div class="row row-cards">
                        <div class="col-md-4">
                            <label for="" class="form-label">关键词</label>
                            <input type="text" class="form-control" name="keywords">
                        </div>
                        <div class="col-md-4">
                            <label for="" class="form-label">导出数量</label>
                            <input type="number" min="0" class="form-control" name="count">
                            <small> >=0 则导出所有 </small>
                        </div>
                        <div class="col-mb-4"></div>
                        <div class="col-mb-4">
                            <button class="btn btn-light" type="submit">立即导出</button>
                        </div>
                    </div>
                </form>
            </div>
        </div>
    </div>
@endsection

@section('scripts')
    <script src="{{mix('plugins/Core/js/admin.js')}}"></script>
@endsection