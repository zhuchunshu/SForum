@extends("app")
@section('content')
    <div class="col-md-12">
        <div class="card">
            <div class="card-body">
                <h3 class="card-title">
                    Hello World
                </h3>
                <form action="" method="POST">
                    <div class="mb-3">
                        <label class="form-label">Text</label>
                        <x-csrf/>
                        <input type="text" class="form-control" name="example-text-input" placeholder="Input placeholder">
                      </div>
                    <button class="btn" type="submit">提交</button>
                </form>
            </div>
        </div>
    </div>
@endsection
