<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class UpdateShortcodePaidPostTable extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::table('shortcode_paid_post', function (Blueprint $table) {
            $table->float('amount')->default(0)->comment('付款金额')->change();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('shortcode_paid_post', function (Blueprint $table) {
            //
        });
    }
}
