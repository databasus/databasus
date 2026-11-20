// The file is served with Content-Disposition, so following the link downloads it and leaves the
// page in place.
export const followFileLink = (url: string, fileName: string) => {
  const link = document.createElement('a');
  link.href = url;
  link.download = fileName;
  link.style.display = 'none';

  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
};
