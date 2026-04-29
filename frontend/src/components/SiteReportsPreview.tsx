import { memo } from 'react';
import { SiteReport } from '../types';

interface Props {
  siteReports: SiteReport[];
}

function SiteReportsPreviewImpl({ siteReports }: Props) {
  if (siteReports.length === 0) return null;
  return (
    <div className="bg-blue-50 rounded-xl border border-blue-200 p-4 space-y-3">
      <div className="text-xs font-semibold text-blue-900">
        사이트 보고서 (본사 슬라이드 뒤에 자동 추가, 편집 없이 그대로 출력)
      </div>
      <div className="space-y-3">
        {siteReports.map((sr) => (
          <SiteReportCard key={sr.id} sr={sr} />
        ))}
      </div>
    </div>
  );
}

function SiteReportCard({ sr }: { sr: SiteReport }) {
  return (
    <div className="bg-white rounded border border-blue-100 p-3">
      <div className="flex flex-wrap gap-3 text-xs text-neutral-600 mb-2 pb-2 border-b border-neutral-100">
        <div><span className="font-medium text-neutral-500">프로젝트:</span> {sr.project_name}</div>
        <div><span className="font-medium text-neutral-500">보고일자:</span> {sr.report_date_text || sr.report_date}</div>
        <div><span className="font-medium text-neutral-500">작성자:</span> {sr.author_names.join(', ')}</div>
      </div>
      {sr.this_week.length > 0 ? (
        <div className="mb-2">
          <div className="text-xs font-medium text-neutral-700 mb-1">금주실적</div>
          <table className="w-full text-[11px] border border-neutral-200">
            <thead className="bg-neutral-50">
              <tr>
                <th className="border border-neutral-200 px-1.5 py-1 text-left">계획업무</th>
                <th className="border border-neutral-200 px-1.5 py-1 text-left w-[8%]">소요일</th>
                <th className="border border-neutral-200 px-1.5 py-1 text-left w-[10%]">시작일</th>
                <th className="border border-neutral-200 px-1.5 py-1 text-left w-[10%]">완료일</th>
                <th className="border border-neutral-200 px-1.5 py-1 text-left w-[8%]">실적</th>
              </tr>
            </thead>
            <tbody>
              {sr.this_week.map((row, i) => (
                <tr key={i} className="align-top">
                  <td className="border border-neutral-200 px-1.5 py-1 whitespace-pre-wrap font-mono">{row.title}</td>
                  <td className="border border-neutral-200 px-1.5 py-1">{row.elapsed_days}</td>
                  <td className="border border-neutral-200 px-1.5 py-1">{row.start_date}</td>
                  <td className="border border-neutral-200 px-1.5 py-1">{row.due_date}</td>
                  <td className="border border-neutral-200 px-1.5 py-1">{row.progress}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
      {sr.next_week.length > 0 ? (
        <div className="mb-2">
          <div className="text-xs font-medium text-neutral-700 mb-1">차주계획</div>
          <table className="w-full text-[11px] border border-neutral-200">
            <thead className="bg-neutral-50">
              <tr>
                <th className="border border-neutral-200 px-1.5 py-1 text-left">계획업무</th>
                <th className="border border-neutral-200 px-1.5 py-1 text-left w-[15%]">시작예정일</th>
                <th className="border border-neutral-200 px-1.5 py-1 text-left w-[15%]">완료예정일</th>
              </tr>
            </thead>
            <tbody>
              {sr.next_week.map((row, i) => (
                <tr key={i} className="align-top">
                  <td className="border border-neutral-200 px-1.5 py-1 whitespace-pre-wrap font-mono">{row.title}</td>
                  <td className="border border-neutral-200 px-1.5 py-1">{row.start_date}</td>
                  <td className="border border-neutral-200 px-1.5 py-1">{row.due_date}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
      {sr.notes ? (
        <div>
          <div className="text-xs font-medium text-neutral-700 mb-1">특이사항</div>
          <pre className="text-[11px] text-neutral-700 whitespace-pre-wrap font-mono bg-neutral-50 p-2 rounded">{sr.notes}</pre>
        </div>
      ) : null}
    </div>
  );
}

const SiteReportsPreview = memo(SiteReportsPreviewImpl);
export default SiteReportsPreview;
