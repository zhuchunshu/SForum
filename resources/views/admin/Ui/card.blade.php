<div id="{{ $id }}" class="card {!! $AddClass !!}">
    @if(!$title)
        <div class="card-body">
            {!! $content !!}
        </div>
    @else
    @if($titleType==0)
        <div class="card-header"><h3 class="card-title">{!! $title !!}</h3></div>
        <div class="card-body">
            {!! $content !!}
        </div>
    @else
    <div class="card-body">
        <h3 class="card-title">
            {!! $title !!}
        </h3>
        {!! $content !!}
    </div>
    @endif
    @endif
</div>