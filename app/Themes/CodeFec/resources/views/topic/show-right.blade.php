<div class="row row-cards">
    @foreach(Itf()->get("show_right") as $value)
        @include($value)
    @endforeach
</div>